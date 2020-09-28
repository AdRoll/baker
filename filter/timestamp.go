package filter

import (
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/AdRoll/baker"
)

var TimestampDesc = baker.FilterDesc{
	Name:   "Timestamp",
	New:    NewTimestamp,
	Config: &FilterTimestampConfig{},
	Help:   "This filter updates the timestamp field to the actual time the line was processed by the pipeline.\n",
}

type FilterTimestampConfig struct {
	Field string // TODO add help string and required field
}

// TODO: rename Timestamp
type FilterTimestamp struct {
	numProcessedLines int64
	numFilteredLines  int64

	fidx baker.FieldIndex
}

func NewTimestamp(cfg baker.FilterParams) (baker.Filter, error) {
	dcfg := cfg.DecodedConfig.(*FilterTimestampConfig)
	fidx, ok := cfg.FieldByName(dcfg.Field)
	if !ok {
		return nil, fmt.Errorf("unknown field %q", dcfg.Field)
	}

	return &FilterTimestamp{fidx: fidx}, nil
}

func (f *FilterTimestamp) Stats() baker.FilterStats {
	return baker.FilterStats{
		NumProcessedLines: atomic.LoadInt64(&f.numProcessedLines),
		NumFilteredLines:  atomic.LoadInt64(&f.numFilteredLines),
	}
}

func (f *FilterTimestamp) Process(l baker.Record, next func(baker.Record)) {
	now := strconv.AppendInt(nil, time.Now().Unix(), 10)
	l.Set(f.fidx, now)
	atomic.AddInt64(&f.numProcessedLines, 1)
	next(l)
}
