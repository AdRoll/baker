// +build !race

package baker_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/AdRoll/baker"
	"github.com/AdRoll/baker/input/inputtest"
	"github.com/AdRoll/baker/output/outputtest"
)

func TestSeparatorDefault(t *testing.T) {
	toml := `
[input]
name="LogLine"
[output]
name="Recorder"
procs=1
fields=["f2", "f0", "f1", "f3"]
`
	runTopology(t, toml, ",")
}

func TestSeparatorComma(t *testing.T) {
	toml := `
[csv]
field_separator='2c' # comma
[input]
name="LogLine"
[output]
name="Recorder"
procs=1
fields=["f2", "f0", "f1", "f3"]
`
	runTopology(t, toml, ",")
}

func TestSeparatorDot(t *testing.T) {
	toml := `
[csv]
field_separator='2e' # dot
[input]
name="LogLine"
[output]
name="Recorder"
procs=1
fields=["f2", "f0", "f1", "f3"]
`
	runTopology(t, toml, ".")
}

func runTopology(t *testing.T, toml, sep string) {
	t.Helper()
	c := baker.Components{
		Inputs:  []baker.InputDesc{inputtest.LogLineDesc},
		Outputs: []baker.OutputDesc{outputtest.RecorderDesc},
		FieldByName: func(name string) (baker.FieldIndex, bool) {
			switch name {
			case "f0":
				return 0, true
			case "f1":
				return 1, true
			case "f2":
				return 2, true
			case "f3":
				return 3, true
			default:
				return 0, false
			}
		},
	}

	cfg, err := baker.NewConfigFromToml(strings.NewReader(toml), c)
	if err != nil {
		t.Fatal(err)
	}

	topology, err := baker.NewTopologyFromConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}

	in := topology.Input.(*inputtest.LogLine)
	out := topology.Output[0].(*outputtest.Recorder)
	r, _ := utf8.DecodeRuneInString(sep)
	for i := 0; i < 10; i++ {
		ll := &baker.LogLine{
			FieldSeparator: byte(r),
		}
		ll.Parse([]byte(fmt.Sprintf("v0%sv1%s%sv3", sep, sep, sep)), nil)
		if !bytes.Equal(ll.Get(0), []byte("v0")) {
			t.Fatalf("parse error, %v", ll.Get(0))
		}
		if !bytes.Equal(ll.Get(1), []byte("v1")) {
			t.Fatalf("parse error, %v", ll.Get(1))
		}
		if !bytes.Equal(ll.Get(2), []byte("")) {
			t.Fatalf("parse error, %v", ll.Get(2))
		}
		if !bytes.Equal(ll.Get(3), []byte("v3")) {
			t.Fatalf("parse error, %v", ll.Get(3))
		}
		in.Lines = append(in.Lines, ll)
	}

	topology.Start()
	topology.Wait()

	if len(out.Records) != 10 {
		t.Fatalf("number of log lines and set of fields should be 3, got %d", len(out.Records))
	}

	for _, lldata := range out.Records {
		// lldata.Fields keep the same order as in output.fields (in TOML)
		fields := lldata.Fields
		t.Errorf("%v", lldata.Record)
		if fields[0] != "" {
			t.Errorf("got fields[0] = %q, want %q", fields[0], "")
		}
		if fields[1] != "v0" {
			t.Errorf("got fields[1] = %q, want %q", fields[1], "v0")
		}
		if fields[2] != "v1" {
			t.Errorf("got fields[2] = %q, want %q", fields[2], "v1")
		}
		if fields[3] != "v3" {
			t.Errorf("got fields[3] = %q, want %q", fields[3], "v3")
		}

		if len(fields) != 4 {
			t.Errorf("got %d fields, want %d", len(fields), 4)
		}
	}
}
