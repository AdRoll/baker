package baker_test

import (
	"strings"
	"testing"

	"github.com/AdRoll/baker"
	"github.com/AdRoll/baker/input/inputtest"
	"github.com/AdRoll/baker/output/outputtest"
)

func TestRawOutputFields(t *testing.T) {

	toml := `
[input]
name="Records"

[output]
name="Recorder"
procs=1
fields=["field2", "field0", "field1", "field3"]
`
	c := baker.Components{
		Inputs:  []baker.InputDesc{inputtest.RecordsDesc},
		Outputs: []baker.OutputDesc{outputtest.RecorderDesc},
		FieldByName: func(name string) (baker.FieldIndex, bool) {
			switch name {
			case "field0":
				return 0, true
			case "field1":
				return 1, true
			case "field2":
				return 2, true
			case "field3":
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

	in := topology.Input.(*inputtest.Records)
	out := topology.Output[0].(*outputtest.Recorder)

	for i := 0; i < 10; i++ {
		ll := baker.LogLine{FieldSeparator: 44}
		ll.Set(0, []byte("value0"))
		ll.Set(1, []byte("value1"))
		ll.Set(3, []byte("value3"))
		in.Records = append(in.Records, &ll)
	}

	topology.Start()
	topology.Wait()

	if len(out.Records) != 10 {
		t.Fatalf("number of records and set of fields should be 3, got %d", len(out.Records))
	}

	for _, lldata := range out.Records {
		// lldata.Fields keep the same order as in output.fields (in TOML)
		fields := lldata.Fields

		if fields[0] != "" {
			t.Errorf("got fields[0] = %q, want %q", fields[0], "")
		}
		if fields[1] != "value0" {
			t.Errorf("got fields[1] = %q, want %q", fields[1], "value0")
		}
		if fields[2] != "value1" {
			t.Errorf("got fields[2] = %q, want %q", fields[2], "value2")
		}
		if fields[3] != "value3" {
			t.Errorf("got fields[3] = %q, want %q", fields[3], "value3")
		}

		if len(fields) != 4 {
			t.Errorf("got %d fields, want %d", len(fields), 4)
		}
	}
}
