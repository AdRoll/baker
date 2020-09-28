package filter

import (
	"strconv"
	"sync/atomic"
	"time"

	"github.com/AdRoll/baker"
	"github.com/SemanticSugar/baker/forklift"
)

var TimestampDesc = baker.FilterDesc{
	Name:   "Timestamp",
	New:    NewTimestamp,
	Config: &FilterTimestampConfig{},
	Help:   "This filter updates the timestamp field to the actual time the line was processed by the pipeline.\n",
}

type FilterTimestampConfig struct{}

type FilterTimestamp struct {
	numProcessedLines int64
	numFilteredLines  int64
}

func NewTimestamp(cfg baker.FilterParams) (baker.Filter, error) {
	return &FilterTimestamp{}, nil
}

func (f *FilterTimestamp) Stats() baker.FilterStats {
	return baker.FilterStats{
		NumProcessedLines: atomic.LoadInt64(&f.numProcessedLines),
		NumFilteredLines:  atomic.LoadInt64(&f.numFilteredLines),
	}
}

func (f *FilterTimestamp) Process(l baker.Record, next func(baker.Record)) {
	now := strconv.AppendInt(nil, time.Now().Unix(), 10)
	l.Set(forklift.FieldTimestamp, now)
	atomic.AddInt64(&f.numProcessedLines, 1)
	next(l)
}
