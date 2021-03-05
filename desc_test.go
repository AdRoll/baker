package baker_test

import (
	"strings"
	"testing"
	"time"

	"github.com/AdRoll/baker"
)

func checkDecodedConfig(t *testing.T, dcfg interface{}) {
	t.Helper()
	if dcfg == nil {
		t.Errorf("DecodedConfig is nil, want struct{}{}")
	}
	_, ok := dcfg.(*struct{})
	if !ok {
		t.Errorf("config cast error")
	}
}

func TestCompDescEmptyConfig(t *testing.T) {

	dummyInputDesc := baker.InputDesc{
		Name: "dummyInput",
		New: func(cfg baker.InputParams) (baker.Input, error) {
			checkDecodedConfig(t, cfg.DecodedConfig)
			return dummyInput{}, nil
		},
		Config: &struct{}{},
	}
	dummyFilterDesc := baker.FilterDesc{
		Name: "dummyFilter",
		New: func(cfg baker.FilterParams) (baker.Filter, error) {
			checkDecodedConfig(t, cfg.DecodedConfig)
			return dummyFilter{}, nil
		},
		Config: &struct{}{},
	}
	dummyOutputDesc := baker.OutputDesc{
		Name: "dummyOutput",
		New: func(cfg baker.OutputParams) (baker.Output, error) {
			checkDecodedConfig(t, cfg.DecodedConfig)
			return dummyOutput{}, nil
		},
		Config: &struct{}{},
	}
	dummyUploadDesc := baker.UploadDesc{
		Name: "dummyUpload",
		New: func(cfg baker.UploadParams) (baker.Upload, error) {
			checkDecodedConfig(t, cfg.DecodedConfig)
			return dummyUpload{}, nil
		},
		Config: &struct{}{},
	}
	dummyMetricsDesc := baker.MetricsDesc{
		Name: "dummyMetric",
		New: func(cfg interface{}) (baker.MetricsClient, error) {
			checkDecodedConfig(t, cfg)
			return dummyMetric{}, nil
		},
		Config: &struct{}{},
	}

	toml := `
[fields]
names = ["field0", "field1", "field2", "field3"]

[input]
name = "dummyInput"

[[filter]]
name = "dummyFilter"

[output]
name = "dummyOutput"
procs = 1
fields = ["field2", "field0", "field1", "field3"]

[upload]
name = "dummyUpload"

[metrics]
name = "dummyMetric"
`
	comp := baker.Components{
		Inputs:  []baker.InputDesc{dummyInputDesc},
		Filters: []baker.FilterDesc{dummyFilterDesc},
		Outputs: []baker.OutputDesc{dummyOutputDesc},
		Uploads: []baker.UploadDesc{dummyUploadDesc},
		Metrics: []baker.MetricsDesc{dummyMetricsDesc},
	}
	cfg, err := baker.NewConfigFromToml(strings.NewReader(toml), comp)
	if err != nil {
		t.Fatalf("cannot parse config: %v", err)
	}

	_, err = baker.NewTopologyFromConfig(cfg)
	if err != nil {
		t.Fatalf("cannot build topology: %v", err)
	}
}

type dummyInput struct{}

func (dummyInput) Run(chan<- *baker.Data) error { return nil }
func (dummyInput) Stop()                        {}
func (dummyInput) Stats() baker.InputStats      { return baker.InputStats{} }
func (dummyInput) FreeMem(*baker.Data)          {}

type dummyFilter struct{}

func (dummyFilter) Process(baker.Record, func(baker.Record)) {}
func (dummyFilter) Stats() baker.FilterStats                 { return baker.FilterStats{} }

type dummyOutput struct{}

func (dummyOutput) Run(<-chan baker.OutputRecord, chan<- string) error { return nil }
func (dummyOutput) Stats() baker.OutputStats                           { return baker.OutputStats{} }
func (dummyOutput) CanShard() bool                                     { return false }

type dummyUpload struct{}

func (dummyUpload) Run(upch <-chan string) error { return nil }
func (dummyUpload) Stop()                        {}
func (dummyUpload) Stats() baker.UploadStats     { return baker.UploadStats{} }

type dummyMetric struct{}

func (dummyMetric) Gauge(string, float64)                            {}
func (dummyMetric) GaugeWithTags(string, float64, []string)          {}
func (dummyMetric) RawCount(string, int64)                           {}
func (dummyMetric) RawCountWithTags(string, int64, []string)         {}
func (dummyMetric) DeltaCount(string, int64)                         {}
func (dummyMetric) DeltaCountWithTags(string, int64, []string)       {}
func (dummyMetric) Histogram(string, float64)                        {}
func (dummyMetric) HistogramWithTags(string, float64, []string)      {}
func (dummyMetric) Duration(string, time.Duration)                   {}
func (dummyMetric) DurationWithTags(string, time.Duration, []string) {}
