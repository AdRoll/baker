package baker_test

import (
	"bytes"
	"flag"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/AdRoll/baker"
	"github.com/AdRoll/baker/filter/filtertest"
	"github.com/AdRoll/baker/input/inputtest"
	"github.com/AdRoll/baker/output"
	"github.com/AdRoll/baker/output/outputtest"
	"github.com/AdRoll/baker/testutil"
	"github.com/AdRoll/baker/upload/uploadtest"
)

type statsInput struct{ inputtest.Base }

func (statsInput) Stats() baker.InputStats {
	bag := baker.MetricsBag{}
	bag.AddDeltaCounter("delta_counter", 1)
	bag.AddGauge("gauge", math.Pi)
	bag.AddHistogram("hist", []float64{1, 2, 3})
	bag.AddTimings("timings", []time.Duration{1 * time.Second, 10 * time.Second, 100 * time.Second})

	return baker.InputStats{
		NumProcessedLines: 93,
		CustomStats:       map[string]string{"k1": "v1", "k2": "v2"},
		Metrics:           bag,
	}
}

type statsFilter struct{ filtertest.Base }

func (statsFilter) Stats() baker.FilterStats {
	bag := baker.MetricsBag{}
	bag.AddDeltaCounter("delta_counter", 3)
	bag.AddGauge("gauge", math.Pi*2)
	bag.AddHistogram("hist", []float64{4, 5, 6, 7})
	bag.AddTimings("timings", []time.Duration{1 * time.Minute, 10 * time.Minute, 100 * time.Minute})

	return baker.FilterStats{
		NumProcessedLines: 67,
		NumFilteredLines:  7,
		Metrics:           bag,
	}
}

type statsOutput struct{ outputtest.Base }

func (statsOutput) Stats() baker.OutputStats {
	bag := baker.MetricsBag{}
	bag.AddDeltaCounter("delta_counter", 7)
	bag.AddGauge("gauge", math.Pi*3)
	bag.AddHistogram("hist", []float64{8, 9, 10, 11})
	bag.AddTimings("timings", []time.Duration{1 * time.Hour, 10 * time.Hour, 100 * time.Hour})

	return baker.OutputStats{
		NumProcessedLines: 53,
		NumErrorLines:     7,
		CustomStats:       map[string]string{"k3": "v3", "k4": "v4"},
		Metrics:           bag,
	}
}

type statsUpload struct{ uploadtest.Base }

func (statsUpload) Stats() baker.UploadStats {
	bag := baker.MetricsBag{}
	bag.AddDeltaCounter("delta_counter", 9)
	bag.AddGauge("gauge", math.Pi*4)
	bag.AddHistogram("hist", []float64{8, 9, 10, 11})
	bag.AddTimings("timings", []time.Duration{1 * time.Microsecond, 10 * time.Microsecond, 100 * time.Microsecond})

	return baker.UploadStats{
		NumProcessedFiles: 17,
		NumErrorFiles:     3,
		CustomStats:       map[string]string{"k5": "v5", "k6": "v6", "k7": "v7"},
		Metrics:           bag,
	}
}

func TestStatsDumper(t *testing.T) {
	buf := &bytes.Buffer{}

	tp := &baker.Topology{
		Input:   &statsInput{},
		Filters: []baker.Filter{&statsFilter{}, &statsFilter{}, &statsFilter{}},
		Output:  []baker.Output{&statsOutput{}, &statsOutput{}},
		Upload:  &statsUpload{},
	}
	sd := baker.NewStatsDumper(tp)
	sd.SetWriter(buf)

	stop := sd.Run()
	// StatsDumper does not print anything the first second
	time.Sleep(1050 * time.Millisecond)
	stop()

	flag.Parse()
	golden := filepath.Join("testdata", t.Name()+".golden")
	if *testutil.UpdateGolden {
		ioutil.WriteFile(golden, buf.Bytes(), os.ModePerm)
		t.Logf("updated: %q", golden)
	}

	testutil.DiffWithGolden(t, buf.Bytes(), golden)
}

func TestStatsDumperInvalidRecords(t *testing.T) {
	// This test contols the correct integration of the StatsDumper with the
	// Topology  by counting the number of invalid fields, that is the number
	// of fields which do not pass the user-specificed validation function, as
	// reported by the StatsDumper, after the topology has finished its execution.
	toml := `
[input]
name="logline"

[output]
name="nop"
fields=["field0"]
`

	components := baker.Components{
		Inputs:  []baker.InputDesc{inputtest.LogLineDesc},
		Outputs: []baker.OutputDesc{output.NopDesc},
		FieldByName: func(n string) (baker.FieldIndex, bool) {
			switch n {
			case "field0":
				return 0, true
			case "field1":
				return 1, true
			}
			return 0, false
		},
		FieldName: func(f baker.FieldIndex) string {
			switch f {
			case 0:
				return "field0"
			case 1:
				return "field1"
			}
			return ""
		},
		Validate: func(r baker.Record) (bool, baker.FieldIndex) {
			// field at index 0 must be "value0"
			// field at index 1 must be "value1"
			if !bytes.Equal(r.Get(0), []byte("value0")) {
				return false, 0
			}
			if !bytes.Equal(r.Get(1), []byte("value1")) {
				return false, 1
			}
			return true, 0
		},
	}

	cfg, err := baker.NewConfigFromToml(strings.NewReader(toml), components)
	if err != nil {
		t.Fatal(err)
	}

	topo, err := baker.NewTopologyFromConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}

	csvs := []string{
		"value0,bar",    // field1 invalid
		"value0,bar",    // field1 invalid
		"value0,value1", // both valid
		"foo,value1",    // field0 invalid
		"foo,bar",       // both invalid (when that's the case the first one is reported)
	}

	in := topo.Input.(*inputtest.LogLine)
	in.Lines = make([]*baker.LogLine, len(csvs))
	for i, csv := range csvs {
		ll := &baker.LogLine{FieldSeparator: ','}
		ll.Parse([]byte(csv), nil)
		in.Lines[i] = ll
	}

	buf := &bytes.Buffer{}
	stats := baker.NewStatsDumper(topo)
	stats.SetWriter(buf)
	stop := stats.Run()

	topo.Start()
	topo.Wait()
	if err = topo.Error(); err != nil {
		t.Fatal(err)
	}

	// The StatsDumper needs at least one second to print anything.
	time.Sleep(time.Second)
	stop()

	// Clean the stats dumper output.
	out := strings.Split(strings.TrimSpace(buf.String()), "\n")
	// Only take the last 2 lines (the final ones).
	out = out[len(out)-2:]

	// The Stats line should contain the following string, indicating 6 validation errors
	want := `errors[p:0 i:4 f:0 o:0 u:0]`
	if !strings.Contains(out[0], want) {
		t.Errorf("StatsDumper stats line doesn't contain %q\nline:\n\t%q", out[0], want)
	}

	// The validation errors line should shown 2 errors for each field
	want = `map[field0:2 field1:2]`
	if !strings.Contains(out[1], want) {
		t.Errorf("StatsDumper validation error line doesn't contain %q\nline:\n\t%q", out[1], want)
	}
}
