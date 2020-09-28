package filter

import (
	"strconv"
	"testing"
	"time"

	"github.com/AdRoll/baker"
)

func TestTimestamp(t *testing.T) {
	fieldByName := func(name string) (baker.FieldIndex, bool) {
		if name == "timestamp" {
			return 10, true
		}
		return 0, false
	}

	t.Run("ok", func(t *testing.T) {
		// Init filter
		cfg := baker.FilterParams{
			ComponentParams: baker.ComponentParams{
				DecodedConfig: &FilterTimestampConfig{
					Field: "timestamp",
				},
				FieldByName: fieldByName,
			},
		}

		filter, err := NewTimestamp(cfg)
		if err != nil {
			t.Fatal(err)
		}

		// Run the filter
		ll := &baker.LogLine{FieldSeparator: ','}
		filter.Process(ll, func(baker.Record) {})

		got := ll.Get(10)
		if got == nil {
			t.Fatalf("got l.Get(10) == nil, want unix timestamp")
		}

		// Check an actual timestamp has been set by the filter
		i, err := strconv.ParseInt(string(got), 10, 64)
		if err != nil {
			t.Fatalf("error parsing timestamp: %v", err)
		}

		// Check that the timestamp is a reasonable timestamp with respect to now
		now := time.Now().UTC()
		ts := time.Unix(i, 0).UTC()
		if now.Sub(ts).Seconds() > 1 {
			t.Fatalf(`timestamp if more than 1s in the past: "%v - %v > 1s"`, now, ts)
		}
	})

	t.Run("field error", func(t *testing.T) {
		// Init filter
		cfg := baker.FilterParams{
			ComponentParams: baker.ComponentParams{
				DecodedConfig: &FilterTimestampConfig{
					Field: "foobar",
				},
				FieldByName: fieldByName,
			},
		}

		_, err := NewTimestamp(cfg)
		if err == nil {
			t.Fatalf("got err = nil, want error: unknown field")
		}
	})
}
