package baker_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"sort"
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
	bag.AddRawCounter("input.raw_count", 10)
	bag.AddDeltaCounter("input.delta_counter", 1)
	bag.AddGauge("input.gauge", math.Pi)
	bag.AddHistogram("input.hist", []float64{1, 2, 3})
	bag.AddTimings("input.timings", []time.Duration{1 * time.Second, 10 * time.Second, 100 * time.Second})

	return baker.InputStats{
		NumProcessedLines: 93,
		CustomStats:       map[string]string{"k1": "v1", "k2": "v2"},
		Metrics:           bag,
	}
}

type statsFilter struct{ filtertest.Base }

func (statsFilter) Stats() baker.FilterStats {
	bag := baker.MetricsBag{}
	bag.AddRawCounter("filter.raw_count", 4)
	bag.AddDeltaCounter("filter.delta_counter", 3)
	bag.AddGauge("filter.gauge", math.Pi*2)
	bag.AddHistogram("filter.hist", []float64{4, 5, 6, 7})
	bag.AddTimings("filter.timings", []time.Duration{1 * time.Minute, 10 * time.Minute, 100 * time.Minute})

	return baker.FilterStats{
		NumFilteredLines: 8,
		Metrics:          bag,
	}
}

type statsOutput struct{ outputtest.Base }

func (statsOutput) Stats() baker.OutputStats {
	bag := baker.MetricsBag{}
	bag.AddRawCounter("output.raw_count", 3)
	bag.AddDeltaCounter("output.delta_counter", 7)
	bag.AddGauge("output.gauge", math.Pi*3)
	bag.AddHistogram("output.hist", []float64{8, 9, 10, 11})
	bag.AddTimings("output.timings", []time.Duration{1 * time.Hour, 10 * time.Hour, 100 * time.Hour})

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
	bag.AddRawCounter("upload.raw_count", 12)
	bag.AddDeltaCounter("upload.delta_counter", 9)
	bag.AddGauge("upload.gauge", math.Pi*4)
	bag.AddHistogram("upload.hist", []float64{8, 9, 10, 11})
	bag.AddTimings("upload.timings", []time.Duration{1 * time.Microsecond, 10 * time.Microsecond, 100 * time.Microsecond})

	return baker.UploadStats{
		NumProcessedFiles: 17,
		NumErrorFiles:     3,
		CustomStats:       map[string]string{"k5": "v5", "k6": "v6", "k7": "v7"},
		Metrics:           bag,
	}
}

var _ baker.MetricsClient = &mockMetrics{}

// mockMetrics is a metrics client that stores all single calls made to itself,
// and sort them so that it's easy to compare output in a mechanical way.
type mockMetrics struct {
	buf bytes.Buffer
}

// showMetrics returns the api calls which text representation has the provided
// prefix, or all of them if the prefix is "".
// NOTE: ignore go runtime metrics.
func (m *mockMetrics) showMetrics(prefix string) []string {
	keep := make([]string, 0)
	for _, s := range strings.Split(m.buf.String(), "\n") {
		if len(strings.TrimSpace(s)) != 0 && !strings.Contains(s, "name=runtime.") {
			if len(prefix) == 0 || strings.HasPrefix(s, prefix) {
				keep = append(keep, s)
			}
		}
	}

	sort.Strings(keep)
	return keep
}

func (m *mockMetrics) Gauge(name string, value float64) {
	fmt.Fprintf(&m.buf, "gauge|name=%s|value=%v\n", name, value)
}
func (m *mockMetrics) RawCount(name string, value int64) {
	fmt.Fprintf(&m.buf, "rawcount|name=%s|value=%v\n", name, value)
}
func (m *mockMetrics) DeltaCount(name string, delta int64) {
	fmt.Fprintf(&m.buf, "delta|name=%s|value=%v\n", name, delta)
}
func (m *mockMetrics) Histogram(name string, value float64) {
	fmt.Fprintf(&m.buf, "hist|name=%s|value=%v\n", name, value)
}
func (m *mockMetrics) Duration(name string, value time.Duration) {
	fmt.Fprintf(&m.buf, "duration|name=%s|value=%v\n", name, value)
}

func (m *mockMetrics) GaugeWithTags(name string, value float64, tags []string) {
	for _, t := range tags {
		fmt.Fprintf(&m.buf, "gauge|name=%s|value=%v|tag=%s\n", name, value, t)
	}
}
func (m *mockMetrics) RawCountWithTags(name string, value int64, tags []string) {
	for _, t := range tags {
		fmt.Fprintf(&m.buf, "rawcount|name=%s|value=%v|tag=%s\n", name, value, t)
	}
}
func (m *mockMetrics) DeltaCountWithTags(name string, delta int64, tags []string) {
	for _, t := range tags {
		fmt.Fprintf(&m.buf, "delta|name=%s|value=%v|tag=%s\n", name, delta, t)
	}
}
func (m *mockMetrics) HistogramWithTags(name string, value float64, tags []string) {
	for _, t := range tags {
		fmt.Fprintf(&m.buf, "hist|name=%s|value=%v|tag=%s\n", name, value, t)
	}
}
func (m *mockMetrics) DurationWithTags(name string, value time.Duration, tags []string) {
	for _, t := range tags {
		fmt.Fprintf(&m.buf, "duration|name=%s|value=%v|tag=%s\n", name, value, t)
	}
}
func (m *mockMetrics) Close() error { return nil }

func TestStatsDumper(t *testing.T) {
	// Check that the StatsDumper correctly reports the metrics gathered from
	// the components. The tests check both the reports printed to the standard
	// output and metrics published to the MetricClient.
	toml := `
[input]
name="statsInput"

[[filter]]
name="statsFilter"

[[filter]]
name="statsFilter"

[[filter]]
name="statsFilter"

[output]
name="statsOutput"
procs=2
fields=["field0"]

[upload]
name="statsUpload"

[metrics]
name="MockMetrics"
`
	components := baker.Components{
		Inputs: []baker.InputDesc{{Name: "statsInput",
			Config: &struct{}{},
			New:    func(baker.InputParams) (baker.Input, error) { return statsInput{}, nil },
		}},
		Filters: []baker.FilterDesc{{Name: "statsFilter",
			Config: &struct{}{},
			New:    func(baker.FilterParams) (baker.Filter, error) { return statsFilter{}, nil },
		}},
		Outputs: []baker.OutputDesc{{Name: "statsOutput",
			Config: &struct{}{},
			New:    func(baker.OutputParams) (baker.Output, error) { return statsOutput{}, nil },
		}},
		Uploads: []baker.UploadDesc{{Name: "statsUpload",
			Config: &struct{}{},
			New:    func(baker.UploadParams) (baker.Upload, error) { return statsUpload{}, nil },
		}},
		Metrics: []baker.MetricsDesc{{
			Name:   "MockMetrics",
			Config: &struct{}{},
			New:    func(interface{}) (baker.MetricsClient, error) { return &mockMetrics{}, nil },
		}},
		FieldByName: func(n string) (baker.FieldIndex, bool) { return 0, true },
		FieldNames:  []string{"foo", "bar"},
	}

	cfg, err := baker.NewConfigFromToml(strings.NewReader(toml), components)
	if err != nil {
		t.Fatal(err)
	}

	tp, err := baker.NewTopologyFromConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}

	sd := baker.NewStatsDumper(tp)
	buf := &bytes.Buffer{}
	sd.SetWriter(buf)

	stop := sd.Run()
	stop()

	// Check StatsDumper output. We first what, by default, gets written to
	// standard output.
	golden := filepath.Join("testdata", t.Name()+".stdout.golden")
	if *testutil.UpdateGolden {
		ioutil.WriteFile(golden, buf.Bytes(), os.ModePerm)
		t.Logf("updated: %q", golden)
	}
	testutil.DiffWithGolden(t, buf.Bytes(), golden)

	// We then check the metrics the StatsDumper sent to the configured metrics
	// client.
	mc := tp.Metrics.(*mockMetrics)
	golden = filepath.Join("testdata", t.Name()+".metrics.golden")
	out := []byte(strings.Join(mc.showMetrics(""), "\n"))
	if *testutil.UpdateGolden {
		ioutil.WriteFile(golden, out, os.ModePerm)
		t.Logf("updated: %q", golden)
	}
	testutil.DiffWithGolden(t, out, golden)
}

func TestStatsDumperInvalidRecords(t *testing.T) {
	// This test controls the correct integration of the StatsDumper with the
	// Topology by counting the number of invalid fields, that is the number of
	// fields which do not pass the user-specificed validation function, as
	// reported by the StatsDumper, after the topology has finished its
	// execution.
	toml := `
[input]
name="logline"

[output]
name="nop"
fields=["field0"]

[metrics]
name="MockMetrics"
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
		FieldNames: []string{"field0", "field1"},
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
		Metrics: []baker.MetricsDesc{{
			Name:   "MockMetrics",
			Config: &struct{}{},
			New: func(interface{}) (baker.MetricsClient, error) {
				return &mockMetrics{}, nil
			},
		}},
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

	stop()

	// Clean the stats dumper output.
	out := strings.Split(strings.TrimSpace(buf.String()), "\n")
	// Only take the last 2 lines (the final ones).
	out = out[len(out)-2:]
	// The Stats line should contain the following string, indicating 6 validation errors.
	wantS := `errors[p:0 i:4 f:0 o:0 u:0]`
	if !strings.Contains(out[0], wantS) {
		t.Errorf("StatsDumper stats line doesn't contain %q\nline:\n\t%q", out[0], wantS)
	}
	// The validation errors line should show 2 errors for each field.
	wantS = `map[field0:2 field1:2]`
	if !strings.Contains(out[1], wantS) {
		t.Errorf("StatsDumper validation error line doesn't contain %q\nline:\n\t%q", out[1], wantS)
	}

	// Check published 'error_lines' metrics.
	mc := topo.Metrics.(*mockMetrics)
	got := mc.showMetrics("rawcount|name=error_lines")
	want := []string{
		"rawcount|name=error_lines.field0|value=2",
		"rawcount|name=error_lines.field1|value=2",
		"rawcount|name=error_lines|value=4",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("error lines =\n%+v\nwant =\n%+v", got, want)
	}
}
