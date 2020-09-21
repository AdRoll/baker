package filter

import (
	"testing"

	"github.com/AdRoll/baker"
	"github.com/AdRoll/baker/testutil"
	"github.com/SemanticSugar/baker/forklift"
)

func TestNotNull(t *testing.T) {
	tests := []struct {
		advertisableEid string
		discard         bool // want it discarded?
	}{
		{
			advertisableEid: "", discard: true,
		},
		{
			advertisableEid: "N6SUJAEWLFHHLAIBDPNASB", discard: false,
		},
	}

	var fields map[baker.FieldIndex]string
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			if tt.advertisableEid != "" {
				fields = map[baker.FieldIndex]string{
					forklift.FieldAdvertisableEid: tt.advertisableEid,
				}
			} else {
				fields = map[baker.FieldIndex]string{}
			}
			// Generate the logline
			logline := testutil.NewLogLineFromMap(fields, forklift.LogLineFieldSeparator)
			if ok, fidx := forklift.ValidateLogLine(logline); !ok {
				t.Fatalf("invalid log line at field: %s", forklift.FieldName(fidx))
			}

			// Init filter
			cfg := baker.FilterParams{}
			cfg.DecodedConfig = &NotNullConfig{
				Fields: []string{"advertisable_eid"},
			}
			filter, _ := NewNotNull(cfg)

			// Run the filter
			discarded := true
			filter.Process(logline, func(baker.Record) { discarded = false })

			// Check result
			if discarded != tt.discard {
				if tt.discard {
					t.Errorf("filter kept a line it shouldn't have: %v", tt.advertisableEid)
				} else {
					t.Errorf("filter discarded a line it shouldn't have: %v", tt.advertisableEid)
				}
			}
		})
	}
}
