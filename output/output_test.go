package output

import (
	"fmt"
	"strings"
	"testing"

	"github.com/AdRoll/baker"
	"github.com/AdRoll/baker/input/inputtest"
	"github.com/AdRoll/baker/output/outputtest"
)

func TestNewTopologyFromConfig_out_concurrency(t *testing.T) {
	const src = `
[input]
name="Base"

[output]
name="Base"
procs=%d
[output.config]
SupportConcurrency = %t
`

	tests := []struct {
		name               string
		procs              int
		supportConcurrency bool
		wantErr            bool
	}{
		{
			name:               "no concurrency, 1 proc",
			procs:              1,
			supportConcurrency: false,
			wantErr:            false,
		},
		{
			name:               "concurrency, 1 proc",
			procs:              1,
			supportConcurrency: true,
			wantErr:            false,
		},
		{
			name:               "no concurrency, 2 procs",
			procs:              2,
			supportConcurrency: false,
			wantErr:            true,
		},
		{
			name:               "concurrency, 2 procs",
			procs:              2,
			supportConcurrency: true,
			wantErr:            false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toml := fmt.Sprintf(src, tt.procs, tt.supportConcurrency)

			components := baker.Components{
				Inputs:      []baker.InputDesc{inputtest.BaseDesc},
				Filters:     []baker.FilterDesc{},
				Outputs:     []baker.OutputDesc{outputtest.BaseDesc},
				FieldByName: func(name string) (baker.FieldIndex, bool) { return 0, true },
				Validate:    func(baker.Record) (bool, baker.FieldIndex) { return true, 0 },
			}

			cfg, err := baker.NewConfigFromToml(strings.NewReader(toml), components)
			if err != nil {
				t.Fatal(err)
			}

			_, err = baker.NewTopologyFromConfig(cfg)
			if (err != nil) != tt.wantErr {
				t.Fatal(err)
			}
		})
	}
}
