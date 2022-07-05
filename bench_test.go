package baker_test

import (
	"bytes"
	"fmt"
	"hash/maphash"
	"strings"
	"testing"

	"github.com/AdRoll/baker"
	"github.com/AdRoll/baker/filter"
	"github.com/AdRoll/baker/filter/filtertest"
	"github.com/AdRoll/baker/filter_error_handler"
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
	fieldNames := []string{
		"fielda",
		"fieldb",
		"fieldc",
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
		FieldNames:    fieldNames,
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

		const wantRaw = "hello,world"
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
	fields := make([][]byte, 0, baker.LogLineNumFields)
	for i := 0; i < int(baker.LogLineNumFields); i++ {
		fields = append(fields, []byte("xxxxxxxxxx"))
	}
	md := baker.Metadata{"foo": "bar"}

	nparse := []int{1, 50, 500, 1000, 2000, 3000}
	for _, nparse := range nparse {
		b.Run(fmt.Sprintf("len=%d", nparse), func(b *testing.B) {
			text := []byte(bytes.Join(fields[:nparse], []byte(",")))

			ll := &baker.LogLine{FieldSeparator: ','}

			b.ReportAllocs()
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				ll.Parse(text, md)
			}
			sink = ll
		})
	}
}

func BenchmarkLogLineToTextFromSet(b *testing.B) {
	nset := []int{1, 5, 50, 100, 254}
	for _, nset := range nset {
		b.Run(fmt.Sprintf("set=%d", nset), func(b *testing.B) {
			ll := baker.LogLine{FieldSeparator: baker.DefaultLogLineFieldSeparator}
			for i := 0; i < nset; i++ {
				ll.Set(baker.FieldIndex(i), []byte("xxxxxxxxxx"))
			}

			b.ReportAllocs()
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				_ = ll.ToText(nil)
			}
		})
	}
}
func BenchmarkLogLineToTextFromParse(b *testing.B) {
	fields := make([][]byte, 0, baker.LogLineNumFields)
	for i := 0; i < int(baker.LogLineNumFields); i++ {
		fields = append(fields, []byte("xxxxxxxxxx"))
	}

	nparse := []int{1, 50, 500, 1000, 2000, 3000}
	for _, nparse := range nparse {
		b.Run(fmt.Sprintf("parse=%d", nparse), func(b *testing.B) {
			text := bytes.Join(fields[:nparse], []byte(","))

			ll := baker.LogLine{FieldSeparator: ','}
			ll.Parse(text, nil)

			b.ReportAllocs()
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				_ = ll.ToText(nil)
			}
		})
	}
}
func BenchmarkLogLineToTextFromParseSet(b *testing.B) {
	fields := make([][]byte, 0, baker.LogLineNumFields)
	for i := 0; i < int(baker.LogLineNumFields); i++ {
		fields = append(fields, []byte("xxxxxxxxxx"))
	}

	nparse := []int{100, 1000, 3000}
	nset := []int{5, 50, 100, 254}
	for _, nparse := range nparse {
		for _, nset := range nset {
			b.Run(fmt.Sprintf("parse=%d,set=%d", nparse, nset), func(b *testing.B) {
				text := bytes.Join(fields[:nparse], []byte(","))

				ll := baker.LogLine{FieldSeparator: ','}
				ll.Parse(text, nil)

				for i := 0; i < nset; i++ {
					ll.Set(baker.FieldIndex(i), []byte("newvalue"))
				}

				b.ReportAllocs()
				b.ResetTimer()
				for n := 0; n < b.N; n++ {
					_ = ll.ToText(nil)
				}
			})
		}
	}
}

func BenchmarkLogLineCopy(b *testing.B) {
	b.Run("set=0", func(b *testing.B) {
		ll := &baker.LogLine{FieldSeparator: baker.DefaultLogLineFieldSeparator}
		buf := bytes.Repeat([]byte(`hello,world,,`), 200)
		ll.Parse(buf, nil)

		b.ReportAllocs()

		var cpy baker.Record
		for n := 0; n < b.N; n++ {
			cpy = ll.Copy()
		}

		if !bytes.Equal(ll.ToText(nil), cpy.ToText(nil)) {
			b.Error("copy != original")
		}
	})

	b.Run("set=1", func(b *testing.B) {
		ll := &baker.LogLine{FieldSeparator: baker.DefaultLogLineFieldSeparator}
		buf := bytes.Repeat([]byte(`hello,world,,`), 200)
		ll.Parse(buf, nil)
		ll.Set(0, []byte("foobar"))

		b.ReportAllocs()

		var cpy baker.Record
		for n := 0; n < b.N; n++ {
			cpy = ll.Copy()
		}

		if !bytes.Equal(ll.ToText(nil), cpy.ToText(nil)) {
			b.Error("copy != original")
		}
	})

	b.Run("set=10", func(b *testing.B) {
		ll := &baker.LogLine{FieldSeparator: baker.DefaultLogLineFieldSeparator}
		buf := bytes.Repeat([]byte(`hello,world,,`), 200)
		ll.Parse(buf, nil)
		ll.Set(0, []byte("foobar"))
		ll.Set(2, []byte("foobar"))
		ll.Set(3, []byte("foobar"))
		ll.Set(6, []byte("foobar"))
		ll.Set(30, []byte("foobar"))
		ll.Set(79, []byte("foobar"))
		ll.Set(124, []byte("foobar"))
		ll.Set(189, []byte("foobar"))
		ll.Set(234, []byte("foobar"))
		ll.Set(798, []byte("foobar"))

		b.ReportAllocs()

		var cpy baker.Record
		for n := 0; n < b.N; n++ {
			cpy = ll.Copy()
		}

		if !bytes.Equal(ll.ToText(nil), cpy.ToText(nil)) {
			b.Error("copy != original")
		}
	})

	b.Run("set=20", func(b *testing.B) {
		ll := &baker.LogLine{FieldSeparator: baker.DefaultLogLineFieldSeparator}
		buf := bytes.Repeat([]byte(`hello,world,,`), 200)
		ll.Parse(buf, nil)
		ll.Set(0, []byte("foobar"))
		ll.Set(2, []byte("foobar"))
		ll.Set(3, []byte("foobar"))
		ll.Set(6, []byte("foobar"))
		ll.Set(30, []byte("foobar"))
		ll.Set(79, []byte("foobar"))
		ll.Set(124, []byte("foobar"))
		ll.Set(189, []byte("foobar"))
		ll.Set(234, []byte("foobar"))
		ll.Set(798, []byte("foobar"))

		ll.Set(801, []byte("foobar"))
		ll.Set(810, []byte("foobar"))
		ll.Set(888, []byte("foobar"))
		ll.Set(902, []byte("foobar"))
		ll.Set(1000, []byte("foobar"))
		ll.Set(1001, []byte("foobar"))
		ll.Set(1002, []byte("foobar"))
		ll.Set(1200, []byte("foobar"))
		ll.Set(1356, []byte("foobar"))
		ll.Set(1789, []byte("foobar"))

		b.ReportAllocs()

		var cpy baker.Record
		for n := 0; n < b.N; n++ {
			cpy = ll.Copy()
		}

		if !bytes.Equal(ll.ToText(nil), cpy.ToText(nil)) {
			b.Error("copy != original")
		}
	})
}

func benchmarkFilterChain(b *testing.B, filterToml string, recordStr []string) {
	const src = `
[fields]
names = ["fielda", "fieldb", "fieldc"]

[filterchain]
procs=1

[input]
name="records"

# insert filter here
%s

[output]
name="discard"
procs=1
fields = ["fielda", "fieldb", "fieldc"]
`
	toml := fmt.Sprintf(src, filterToml)

	components := baker.Components{
		Inputs:              []baker.InputDesc{inputtest.RecordsDesc},
		Filters:             filter.All,
		FilterErrorHandlers: filter_error_handler.All,
		Outputs:             []baker.OutputDesc{discardOutputDesc},
	}

	cfg, err := baker.NewConfigFromToml(strings.NewReader(toml), components)
	if err != nil {
		b.Fatal(err)
	}

	const nlines = 10000
	lines := make([]baker.Record, nlines)
	for i := 0; i < nlines; i++ {
		l := &baker.LogLine{FieldSeparator: baker.DefaultLogLineFieldSeparator}
		if err := l.Parse([]byte(recordStr[i%len(recordStr)]), nil); err != nil {
			b.Fatal(err)
		}
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
	}
}

// discardOutputDesc describes the discard output.
var discardOutputDesc = baker.OutputDesc{
	Name:   "discard",
	New:    func(baker.OutputParams) (baker.Output, error) { return &discardOutput{}, nil },
	Config: &struct{}{},
	Raw:    false,
}

type discardOutput struct {
	outputtest.Base
}

func (*discardOutput) Run(ch <-chan baker.OutputRecord, _ chan<- string) error {
	for range ch {
	}
	return nil
}

func BenchmarkFilterChain(b *testing.B) {
	b.Run("url-escape", func(b *testing.B) {
		toml := `
		[[filter]]
		name="urlescape"
		
		[filter.config]
		srcfield="fielda"
		dstfield="fieldb"
		`
		// Simple 'pure' filter: url-escapes a field, writes the result in another.
		// No error, no possibility of drop.
		benchmarkFilterChain(b, toml, []string{
			"foo,bar,baz",
		})
	})

	b.Run("url-unescape", func(b *testing.B) {
		toml := `
		[[filter]]
		name="urlescape"
		dropOnError=true
		
		[filter.config]
		srcfield="fielda"
		dstfield="fieldb"
		unescape = true

		[[filter.error_handler]]
		name = "ClearFields"
		[filter.error_handler.config]
		fields = ["fieldb"]
		`
		// Now the unescaping may fail, in which case the destination is cleared.
		// No possibility of drop.
		benchmarkFilterChain(b, toml, []string{
			"%zzzzz,bar,baz", // fielda is not well-formed, it can't be 'unescaped'
			"%6F,bar,baz",    // fielda is well formed 'url escaped'
		})
	})

	b.Run("not-null/drop", func(b *testing.B) {
		toml := `
		[[filter]]
		name="notnull"
		dropOnError=true
		
		[filter.config]
		fields= ["fielda"]
		`
		// NotNull filter drops a record if a field is null/empty.
		benchmarkFilterChain(b, toml, []string{
			",bar,baz", // fielda is empty, the record will be dropped
		})
	})

	b.Run("not-null/drop20%", func(b *testing.B) {
		toml := `
		[[filter]]
		name="notnull"
		dropOnError=true
		
		[filter.config]
		fields= ["fielda"]
		`
		// NotNull filter drops a record if a field is null/empty.
		benchmarkFilterChain(b, toml, []string{
			strings.Repeat("foo,bar,baz", 4), // not dropped
			",bar,baz",                       // fielda is empty -> record is dropped 20% of the time
		})
	})

	b.Run("not-null/drop2%", func(b *testing.B) {
		toml := `
		[[filter]]
		name="notnull"
		dropOnError=true
		
		[filter.config]
		fields= ["fielda"]
		`
		// NotNull filter drops a record if a field is null/empty.
		benchmarkFilterChain(b, toml, []string{
			strings.Repeat("foo,bar,baz", 49), // not dropped
			",bar,baz",                        // fielda is empty -> record is dropped 2% of the time
		})
	})
}
