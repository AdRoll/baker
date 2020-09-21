package filter

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/AdRoll/baker"
	"github.com/AdRoll/baker/testutil"
	"github.com/SemanticSugar/baker/forklift"
)

func TestTimestampRange(t *testing.T) {
	tests := []struct {
		desc           string
		fieldTimestamp string
		startDate      string
		endDate        string
		discarded      bool
	}{
		{
			desc:           "Valid",
			fieldTimestamp: "1580981641", // 2020-02-06 09:34:01
			startDate:      "2019-02-06 09:34:01",
			endDate:        "2022-02-06 09:34:01",
			discarded:      false,
		},
		{
			desc:           "Too old",
			fieldTimestamp: "1528277641", // 2018-06-06 09:34:01
			startDate:      "2019-02-06 09:34:01",
			endDate:        "2022-02-06 09:34:01",
			discarded:      true,
		},
		{
			desc:           "Too new",
			fieldTimestamp: "1565084041", // 2019-08-06 09:34:01
			startDate:      "2017-02-06 09:34:01",
			endDate:        "2018-02-06 09:34:01",
			discarded:      true,
		},
		{
			desc:           "Inclusive start",
			fieldTimestamp: "1486373641", // 2017-02-06 09:34:01
			startDate:      "2017-02-06 09:34:01",
			endDate:        "2018-02-06 09:34:01",
			discarded:      false,
		},
		{
			desc:           "Exclusive end",
			fieldTimestamp: "1517909641", // 2018-02-06 09:34:01
			startDate:      "2017-02-06 09:34:01",
			endDate:        "2018-02-06 09:34:01",
			discarded:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			i, err := strconv.ParseInt(tt.fieldTimestamp, 10, 64)
			if err != nil {
				t.Errorf("Cannot convert fieldTimestamp")
			}

			fields := map[baker.FieldIndex]string{
				forklift.FieldTimestamp: fmt.Sprintf("%d", i),
			}
			// Generate the logline
			logline := testutil.NewLogLineFromMap(fields, forklift.LogLineFieldSeparator)
			if ok, fidx := forklift.ValidateLogLine(logline); !ok {
				t.Fatalf("invalid log line at field: %s", forklift.FieldName(fidx))
			}

			// Init filter
			cfg := baker.FilterParams{}
			cfg.DecodedConfig = &TimestampRangeConfig{
				StartDatetime: tt.startDate,
				EndDatetime:   tt.endDate,
			}
			filter, err := NewTimestampRange(cfg)
			if err != nil {
				t.Errorf("error initializing the filter %v", err.Error())
			}

			// Run the filter
			discarded := true
			numDiscardedLines := 1
			filter.Process(logline, func(baker.Record) {
				discarded = false
				numDiscardedLines = 0
			})

			// Check result
			if tt.discarded != discarded {
				t.Errorf("got: %t, want: %t", tt.discarded, discarded)
			}

			s := filter.Stats()
			if s.NumProcessedLines != 1 {
				t.Errorf("got: %d, want: %d", 1, s.NumProcessedLines)
			}
			if s.NumFilteredLines != int64(numDiscardedLines) {
				t.Errorf("got: %d, want: %d", 1, s.NumFilteredLines)
			}
		})
	}
}

func TestNewTimestampRangeErrors(t *testing.T) {
	tests := []struct {
		desc      string
		startDate string
		endDate   string
	}{
		{
			desc:      "empty start",
			startDate: "",
			endDate:   "2022-02-06 09:34:01",
		},
		{
			desc:      "empty end",
			startDate: "2019-02-06 09:34:01",
			endDate:   "",
		},
		{
			desc:      "wrong start",
			startDate: "2017-32-06 09:34:01",
			endDate:   "2018-02-06 09:34:01",
		},
		{
			desc:      "wrong end",
			startDate: "2017-02-06 09:34:01",
			endDate:   "2018-02-06 09:34",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			// Init filter
			cfg := baker.FilterParams{}
			cfg.DecodedConfig = &TimestampRangeConfig{
				StartDatetime: tt.startDate,
				EndDatetime:   tt.endDate,
			}
			_, err := NewTimestampRange(cfg)
			if err == nil {
				t.Errorf("expected error")
			}
		})
	}

	// Empty config must return an error as well
	t.Run("nil config", func(t *testing.T) {
		_, err := NewTimestampRange(baker.FilterParams{})
		if err == nil {
			t.Errorf("expected error")
		}
	})
}
