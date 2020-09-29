package filter

import (
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/AdRoll/baker"
)

// TimestampDesc describes the Timestamp filter.
var TimestampDesc = baker.FilterDesc{
	Name:   "Timestamp",
	New:    NewTimestamp,
	Config: &TimestampConfig{},
	Help:   "Sets a field to the Unix Epoch timestamp at which the record is processed",
}

// TimestampConfig holds configuration paramters for the Timestamp filter.
type TimestampConfig struct {
	Field string `help:"field to set to the unix Epoch timestamp" required:"true"`
}

// Timestamp is a baker filter that sets the value of a certain field to the
// Unix Epoch timestamp (in seconds, UTC) for all the records it processes.
type Timestamp struct {
	nlines int64
	fidx   baker.FieldIndex
}

// NewTimestamp creates and configures a Timestamp filter.
func NewTimestamp(cfg baker.FilterParams) (baker.Filter, error) {
	dcfg := cfg.DecodedConfig.(*TimestampConfig)
	fidx, ok := cfg.FieldByName(dcfg.Field)
	if !ok {
		return nil, fmt.Errorf("unknown field %q", dcfg.Field)
	}

	return &Timestamp{fidx: fidx}, nil
}

// Stats implements baker.Filter.
func (f *Timestamp) Stats() baker.FilterStats {
	return baker.FilterStats{
		NumProcessedLines: atomic.LoadInt64(&f.nlines),
	}
}

// Process implements baker.Filter.
func (f *Timestamp) Process(l baker.Record, next func(baker.Record)) {
	atomic.AddInt64(&f.nlines, 1)

	now := strconv.AppendInt(nil, time.Now().Unix(), 10)
	l.Set(f.fidx, now)

	next(l)
}
