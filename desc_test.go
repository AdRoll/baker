package baker_test

import (
	"strings"
	"testing"

	"github.com/AdRoll/baker"
	"github.com/AdRoll/baker/filter/filtertest"
	"github.com/AdRoll/baker/input/inputtest"
	"github.com/AdRoll/baker/output/outputtest"
	"github.com/AdRoll/baker/upload/uploadtest"
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
			return inputtest.Base{}, nil
		},
		Config: &struct{}{},
	}
	dummyFilterDesc := baker.FilterDesc{
		Name: "dummyFilter",
		New: func(cfg baker.FilterParams) (baker.Filter, error) {
			checkDecodedConfig(t, cfg.DecodedConfig)
			return filtertest.Base{}, nil
		},
		Config: &struct{}{},
	}
	dummyOutputDesc := baker.OutputDesc{
		Name: "dummyOutput",
		New: func(cfg baker.OutputParams) (baker.Output, error) {
			checkDecodedConfig(t, cfg.DecodedConfig)
			return outputtest.Base{}, nil
		},
		Config: &struct{}{},
	}
	dummyUploadDesc := baker.UploadDesc{
		Name: "dummyUpload",
		New: func(cfg baker.UploadParams) (baker.Upload, error) {
			checkDecodedConfig(t, cfg.DecodedConfig)
			return uploadtest.Base{}, nil
		},
		Config: &struct{}{},
	}
	dummyMetricsDesc := baker.MetricsDesc{
		Name: "dummyMetric",
		New: func(cfg interface{}) (baker.MetricsClient, error) {
			checkDecodedConfig(t, cfg)
			return baker.NopMetrics{}, nil
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
