package baker_test

import (
	"bytes"
	"fmt"
	"hash/maphash"
	"strings"
	"testing"

	"github.com/AdRoll/baker"
	"github.com/AdRoll/baker/filter/filtertest"
	"github.com/AdRoll/baker/input/inputtest"
	"github.com/AdRoll/baker/output/outputtest"
)

func BenchmarkTopology_NonRaw_NoSharding(b *testing.B) { benchmarkTopology(b, 1000, false, false) }
func BenchmarkTopology_NonRaw_Sharding(b *testing.B)   { benchmarkTopology(b, 1000, false, true) }
func BenchmarkTopology_Raw_NoSharding(b *testing.B)    { benchmarkTopology(b, 1000, true, false) }
func BenchmarkTopology_Raw_Sharding(b *testing.B)      { benchmarkTopology(b, 1000, true, true) }

func benchmarkTopology(b *testing.B, nlines int, raw, sharding bool) {
	const src = `
[filterchain]
procs=1

[input]
name="records"

[[filter]]
name="passthrough"

[output]
name="%s" # recoder / rawrecorder
procs=1
%s # sharding
fields=["fielda", "fieldb"]
`
	shardingStr := ""
	if sharding {
		shardingStr = `sharding="fielda"`
	}
	outName := "recorder"
	if raw {
		outName = "rawrecorder"
	}
	toml := fmt.Sprintf(src, outName, shardingStr)

	mh := maphash.Hash{}
	hash := func(f int) baker.ShardingFunc {
		return func(l baker.Record) uint64 {
			mh.Reset()
			mh.Write(l.Get(baker.FieldIndex(f)))
			return mh.Sum64()
		}
	}

	var fieldByName = func(name string) (baker.FieldIndex, bool) {
		switch name {
		case "fielda":
			return 0, true
		case "fieldb":
			return 1, true
		}
		return 0, false
	}

	var shardingFuncs = map[baker.FieldIndex]baker.ShardingFunc{
		0: hash(0),
		1: hash(1),
	}

	components := baker.Components{
		Inputs:        []baker.InputDesc{inputtest.RecordsDesc},
		Filters:       []baker.FilterDesc{filtertest.PassThroughDesc},
		Outputs:       []baker.OutputDesc{outputtest.RawRecorderDesc, outputtest.RecorderDesc},
		ShardingFuncs: shardingFuncs,
		FieldByName:   fieldByName,
		Validate:      func(baker.Record) (bool, baker.FieldIndex) { return true, 0 },
	}

	cfg, err := baker.NewConfigFromToml(strings.NewReader(toml), components)
	if err != nil {
		b.Fatal(err)
	}

	lines := make([]baker.Record, nlines)
	for i := 0; i < nlines; i++ {
		l := &baker.LogLine{FieldSeparator: baker.DefaultLogLineFieldSeparator}
		l.Set(0, []byte("hello"))
		l.Set(1, []byte("world"))
		lines[i] = l
	}

	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		// Setup and feed the input component
		topo, err := baker.NewTopologyFromConfig(cfg)
		if err != nil {
			b.Fatal(err)
		}

		in := topo.Input.(*inputtest.Records)
		in.Records = lines

		topo.Start()
		topo.Wait()
		if err := topo.Error(); err != nil {
			b.Fatalf("topology error: %v", err)
		}

		const wantRaw = "hello,world,,"
		for _, ll := range topo.Output[0].(*outputtest.Recorder).Records {
			if ll.Fields[0] != "hello" || ll.Fields[1] != "world" {
				b.Fatalf("ll.Fields[0], ll.Fields[1] = %q, %q, want %q, %q", ll.Fields[0], ll.Fields[1], "hello", "world")
			}
			if raw {
				if !bytes.Equal(ll.Record, []byte(wantRaw)) {
					b.Errorf("ll.Line = %q, want %q", ll.Record, wantRaw)
				}
			}
		}
	}
}

var sink interface{}

func BenchmarkLogLineParse(b *testing.B) {
	var ll baker.Record
	ll = &baker.LogLine{FieldSeparator: baker.DefaultLogLineFieldSeparator}
	buf := bytes.Repeat([]byte(`hello,world,,`), 200)
	md := baker.Metadata{"foo": "bar"}

	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		ll.Parse(buf, md)
	}
	sink = ll
}

func BenchmarkLogLineCopyParsed(b *testing.B) {
	var ll baker.Record
	ll = &baker.LogLine{FieldSeparator: baker.DefaultLogLineFieldSeparator}
	buf := bytes.Repeat([]byte(`hello,world,,`), 200)
	ll.Parse(buf, nil)

	b.ReportAllocs()

	var cpy baker.Record
	for n := 0; n < b.N; n++ {
		cpy = ll.Copy()
	}
	sink = cpy

	if !bytes.Equal(ll.ToText(nil), cpy.ToText(nil)) {
		b.Error("copy != original")
	}
}

func BenchmarkLogLineCopyModified(b *testing.B) {
	var ll baker.Record
	ll = &baker.LogLine{FieldSeparator: baker.DefaultLogLineFieldSeparator}
	buf := bytes.Repeat([]byte(`hello,world,,`), 200)
	ll.Parse(buf, nil)
	ll.Set(0, []byte("foobar"))

	b.ReportAllocs()

	var cpy baker.Record
	for n := 0; n < b.N; n++ {
		cpy = ll.Copy()
	}
	sink = cpy

	if !bytes.Equal(ll.ToText(nil), cpy.ToText(nil)) {
		b.Error("copy != original")
	}
}
