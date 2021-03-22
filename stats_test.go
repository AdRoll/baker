package baker_test

import (
	"bytes"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"reflect"
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

type mockMetrics map[string]interface{}

func (m mockMetrics) Gauge(name string, value float64) {
	if _, ok := m["g:"+name]; !ok {
		m["g:"+name] = []float64{}
	}
	m["g:"+name] = append(m["g:"+name].([]float64), value)
}

func (m mockMetrics) RawCount(name string, value int64) {
	if _, ok := m["c:"+name]; !ok {
		m["c:"+name] = []int64{}
	}
	m["c:"+name] = append(m["c:"+name].([]int64), value)
}
func (m mockMetrics) DeltaCount(name string, delta int64) {
	if _, ok := m["d:"+name]; !ok {
		m["d:"+name] = []int64{}
	}
	m["d:"+name] = append(m["d:"+name].([]int64), delta)
}
func (m mockMetrics) Histogram(name string, value float64) {
	if _, ok := m["h:"+name]; !ok {
		m["h:"+name] = []float64{}
	}
	m["h:"+name] = append(m["h:"+name].([]float64), value)
}
func (m mockMetrics) Duration(name string, value time.Duration) {
	if _, ok := m["t:"+name]; !ok {
		m["t:"+name] = []time.Duration{}
	}
	m["t:"+name] = append(m["t:"+name].([]time.Duration), value)
}

// skip
func (m mockMetrics) GaugeWithTags(name string, value float64, tags []string)          {}
func (m mockMetrics) RawCountWithTags(name string, value int64, tags []string)         {}
func (m mockMetrics) DeltaCountWithTags(name string, delta int64, tags []string)       {}
func (m mockMetrics) HistogramWithTags(name string, value float64, tags []string)      {}
func (m mockMetrics) DurationWithTags(name string, value time.Duration, tags []string) {}

func TestStatsDumper(t *testing.T) {
	// Check that the StatsDumper correctly reports the metrics gathered from the components.
	// The tests check both the reports printed to the standard output and metrics published to the MetricClient.

	wantMetrics := map[string]interface{}{
		// default published metrics
		"c:processed_lines": int64(53 + 53),
		"c:uploads":         int64(17),
		"c:upload_errors":   int64(3),
		"c:error_lines":     int64(0 + 0 + (8 + 8 + 8) + (7 + 7)), // invalid + parseErrors + filtered + outErrors
		"c:filtered_lines":  int64(8 + 8 + 8),

		// custom input metric
		"c:input.raw_count":     int64(10),
		"d:input.delta_counter": int64(1),
		"g:input.gauge":         float64(math.Pi),
		"h:input.hist":          []float64{1, 2, 3},
		"t:input.timings": []time.Duration{
			1 * time.Second, 10 * time.Second, 100 * time.Second,
		},

		// custom filter metric
		"c:filter.raw_count":     int64(4 + 4 + 4),
		"d:filter.delta_counter": int64(3 + 3 + 3),
		"g:filter.gauge":         float64((math.Pi*2 + math.Pi*2 + math.Pi*2) / 3),
		"h:filter.hist": []float64{
			4, 5, 6, 7, // first filter
			4, 5, 6, 7, // second filter
			4, 5, 6, 7, // third filter
		},
		"t:filter.timings": []time.Duration{
			1 * time.Minute, 10 * time.Minute, 100 * time.Minute, // first filter
			1 * time.Minute, 10 * time.Minute, 100 * time.Minute, // second filter
			1 * time.Minute, 10 * time.Minute, 100 * time.Minute, // third filter
		},

		// custom output metrics
		"c:output.raw_count":     int64(3 + 3),
		"d:output.delta_counter": int64(7 + 7),
		"g:output.gauge":         float64((math.Pi*3 + math.Pi*3) / 2),
		"h:output.hist": []float64{
			8, 9, 10, 11, // first output
			8, 9, 10, 11, // second output
		},
		"t:output.timings": []time.Duration{
			1 * time.Hour, 10 * time.Hour, 100 * time.Hour, // first output
			1 * time.Hour, 10 * time.Hour, 100 * time.Hour, // second output
		},

		// custom upload metric
		"c:upload.raw_count":     int64(12),
		"d:upload.delta_counter": int64(9),
		"g:upload.gauge":         float64(math.Pi * 4),
		"h:upload.hist":          []float64{8, 9, 10, 11},
		"t:upload.timings": []time.Duration{
			1 * time.Microsecond, 10 * time.Microsecond, 100 * time.Microsecond,
		},
	}

	tp := &baker.Topology{
		Input:   &statsInput{},
		Filters: []baker.Filter{&statsFilter{}, &statsFilter{}, &statsFilter{}},
		Output:  []baker.Output{&statsOutput{}, &statsOutput{}},
		Upload:  &statsUpload{},
		Metrics: mockMetrics{},
	}

	sd := baker.NewStatsDumper(tp)
	buf := &bytes.Buffer{}
	sd.SetWriter(buf)

	stop := sd.Run()
	// StatsDumper does not print anything the first second
	time.Sleep(1050 * time.Millisecond)
	stop()

	// check std output
	golden := filepath.Join("testdata", t.Name()+".golden")
	if *testutil.UpdateGolden {
		ioutil.WriteFile(golden, buf.Bytes(), os.ModePerm)
		t.Logf("updated: %q", golden)
	}
	testutil.DiffWithGolden(t, buf.Bytes(), golden)

	// check published metrics. The MockMetrics should contain two times the
	// same metrics 1 collected after 1 second and the other after stop
	for k, want := range wantMetrics {
		mc := tp.Metrics.(mockMetrics)
		get, ok := mc[k]
		if !ok {
			t.Errorf("metric %v not found", k)
			continue
		}

		switch k[0] {
		case 'c', 'd':
			w := want.(int64)
			g := get.([]int64)
			if len(g) != 2 || g[0] != w || g[1] != w {
				t.Errorf("metric %v: want %v, get %v", k[2:], w, g[0])
			}
		case 'g':
			w := want.(float64)
			g := get.([]float64)
			if len(g) != 2 || g[0] != w || g[1] != w {
				t.Errorf("metric %v: want %v, get %v", k[2:], w, g[0])
			}
		case 'h':
			w := append(want.([]float64), want.([]float64)...)
			g := get.([]float64)
			if !reflect.DeepEqual(w, g) {
				t.Errorf("metric %v: want %v, get %v", k[2:], w, g)
			}
		case 't':
			w := append(want.([]time.Duration), want.([]time.Duration)...)
			g := get.([]time.Duration)
			if !reflect.DeepEqual(w, g) {
				t.Errorf("metric %v: want %v, get %v", k[2:], w, g)
			}
		default:
			t.Fatalf("wantMetrics map malformed")
		}
	}
}

func TestStatsDumperInvalidRecords(t *testing.T) {
	// This test controls the correct integration of the StatsDumper with the
	// Topology  by counting the number of invalid fields, that is the number
	// of fields which do not pass the user-specificed validation function, as
	// reported by the StatsDumper, after the topology has finished its execution.
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
				return make(mockMetrics, 0), nil
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

	// The StatsDumper needs at least one second to print anything.
	time.Sleep(1050 * time.Millisecond)
	stop()

	// Clean the stats dumper output.
	out := strings.Split(strings.TrimSpace(buf.String()), "\n")
	// Only take the last 2 lines (the final ones).
	out = out[len(out)-2:]
	// The Stats line should contain the following string, indicating 6 validation errors
	wantS := `errors[p:0 i:4 f:0 o:0 u:0]`
	if !strings.Contains(out[0], wantS) {
		t.Errorf("StatsDumper stats line doesn't contain %q\nline:\n\t%q", out[0], wantS)
	}
	// The validation errors line should shown 2 errors for each field
	wantS = `map[field0:2 field1:2]`
	if !strings.Contains(out[1], wantS) {
		t.Errorf("StatsDumper validation error line doesn't contain %q\nline:\n\t%q", out[1], wantS)
	}

	// check published metrics
	mc := topo.Metrics.(mockMetrics)
	wantN := int64(4)
	v, ok := mc["c:error_lines"]
	if !ok || v.([]int64)[0] != wantN || v.([]int64)[1] != wantN {
		t.Errorf("want %v, get %v", wantN, v.([]int64)[0])
	}
	wantN = int64(2)
	v, ok = mc["c:error_lines.field0"]
	if !ok || v.([]int64)[0] != wantN || v.([]int64)[1] != wantN {
		t.Errorf("want %v, get %v", wantN, v.([]int64)[0])
	}
	wantN = int64(2)
	v, ok = mc["c:error_lines.field1"]
	if !ok || v.([]int64)[0] != wantN || v.([]int64)[1] != wantN {
		t.Errorf("want %v, get %v", wantN, v.([]int64)[0])
	}
}
