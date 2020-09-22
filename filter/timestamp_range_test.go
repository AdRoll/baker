package filter

import (
	"strconv"
	"testing"

	"github.com/AdRoll/baker"
)

func TestTimestampRange(t *testing.T) {
	tests := []struct {
		name      string
		ts        int
		startDate string
		endDate   string
		want      bool // true: kept, false: discarded
	}{
		{
			name:      "valid",
			ts:        1580981641, // 2020-02-06 09:34:01
			startDate: "2019-02-06 09:34:01",
			endDate:   "2022-02-06 09:34:01",
			want:      true,
		},
		{
			name:      "past lower bound",
			ts:        1528277641, // 2018-06-06 09:34:01
			startDate: "2019-02-06 09:34:01",
			endDate:   "2022-02-06 09:34:01",
			want:      false,
		},
		{
			name:      "past upper bound",
			ts:        1565084041, // 2019-08-06 09:34:01
			startDate: "2017-02-06 09:34:01",
			endDate:   "2018-02-06 09:34:01",
			want:      false,
		},
		{
			name:      "on lower bound",
			ts:        1486373641, // 2017-02-06 09:34:01
			startDate: "2017-02-06 09:34:01",
			endDate:   "2018-02-06 09:34:01",
			want:      true,
		},
		{
			name:      "on upper bound",
			ts:        1517909641, // 2018-02-06 09:34:01
			startDate: "2017-02-06 09:34:01",
			endDate:   "2018-02-06 09:34:01",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Init filter
			cfg := baker.FilterParams{
				ComponentParams: baker.ComponentParams{
					DecodedConfig: &TimestampRangeConfig{
						StartDatetime: tt.startDate,
						EndDatetime:   tt.endDate,
					},
					FieldByName: func(name string) (baker.FieldIndex, bool) { return 0, true },
				},
			}
			filter, err := NewTimestampRange(cfg)
			if err != nil {
				t.Fatal(err)
			}

			// Run the filter
			kept := false
			ndiscarded := 1

			ll := &baker.LogLine{FieldSeparator: ','}
			ll.Set(0, []byte(strconv.Itoa(tt.ts)))
			filter.Process(ll, func(baker.Record) {
				kept = true
				ndiscarded++
			})

			// Check result
			if kept != tt.want {
				t.Errorf("got record kept=%t, want %t", kept, tt.want)
			}
		})
	}
}

func TestNewTimestampRangeErrors(t *testing.T) {
	tests := []struct {
		desc      string
		startDate string
		endDate   string
		field     string
	}{
		{
			desc:      "invalid lower bound",
			startDate: "2017-32-06 09:34:01",
			endDate:   "2018-02-06 09:34:01",
			field:     "timestamp",
		},
		{
			desc:      "invalid upper bound",
			startDate: "2017-02-06 09:34:01",
			endDate:   "2018-02-06 09:34",
			field:     "timestamp",
		},
		{
			desc:      "unknown field",
			startDate: "2017-02-06 09:34:01",
			endDate:   "2018-02-06 09:34:01",
			field:     "foobar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			// Init filter
			cfg := baker.FilterParams{
				ComponentParams: baker.ComponentParams{
					DecodedConfig: &TimestampRangeConfig{
						StartDatetime: tt.startDate,
						EndDatetime:   tt.endDate,
						Field:         tt.field,
					},
					FieldByName: func(name string) (baker.FieldIndex, bool) {
						if name != "timestamp" {
							return 0, false
						}

						return 0, true
					},
				},
			}

			if _, err := NewTimestampRange(cfg); err == nil {
				t.Errorf("err = nil, want an error")
			}
		})
	}
}
