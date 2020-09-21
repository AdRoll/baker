package filter

import (
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/AdRoll/baker"
	"github.com/SemanticSugar/baker/forklift"
)

// TimestampRangeDesc describes the NotNull filter.
var TimestampRangeDesc = baker.FilterDesc{
	Name:   "TimestampRange",
	New:    NewTimestampRange,
	Config: &TimestampRangeConfig{},
	Help:   "Discard all loglines not included in the provided time range\n",
}

// TimestampRangeConfig holds configuration paramters for the NotNull filter.
type TimestampRangeConfig struct {
	StartDatetime string `help:"The oldest accepted timestamp of the loglines (inclusive, UTC) format:'2006-01-31 15:04:05'" default:""`
	EndDatetime   string `help:"The most recent accepted timestamp of the loglines (exclusive, UTC) format:'2006-01-31 15:04:05'" default:""`
}

// TimestampRange is a baker filter that discards records depending on the
// value of a field representing a Unix timestamp.
type TimestampRange struct {
	numProcessedLines int64
	numFilteredLines  int64
	startDate         int64
	endDate           int64
}

// NewTimestampRange creates and configures a TimestampRange filter.
func NewTimestampRange(cfg baker.FilterParams) (baker.Filter, error) {
	if cfg.DecodedConfig == nil {
		cfg.DecodedConfig = &TimestampRangeConfig{}
	}
	dcfg := cfg.DecodedConfig.(*TimestampRangeConfig)

	if dcfg.StartDatetime == "" || dcfg.EndDatetime == "" {
		return nil, fmt.Errorf("Missing required configurations")
	}

	const timeLayout = "2006-01-02 15:04:05"

	s, err := time.Parse(timeLayout, dcfg.StartDatetime)
	if err != nil {
		return nil, err
	}

	e, err := time.Parse(timeLayout, dcfg.EndDatetime)
	if err != nil {
		return nil, err
	}

	return &TimestampRange{startDate: s.Unix(), endDate: e.Unix()}, nil
}

// Stats implements baker.Filter.
func (f *TimestampRange) Stats() baker.FilterStats {
	return baker.FilterStats{
		NumProcessedLines: atomic.LoadInt64(&f.numProcessedLines),
		NumFilteredLines:  atomic.LoadInt64(&f.numFilteredLines),
	}
}

// Process implements baker.Filter.
func (f *TimestampRange) Process(l baker.Record, next func(baker.Record)) {
	atomic.AddInt64(&f.numProcessedLines, 1)
	// Convert the logline timestamp to unix time (int64)
	ts, err := strconv.ParseInt(string(l.Get(forklift.FieldTimestamp)), 10, 64)

	// All timestamps outside the start-end daterange must be filteres, as well as loglines with
	// unparsable timestamps
	if err != nil || ts < f.startDate || ts >= f.endDate {
		atomic.AddInt64(&f.numFilteredLines, 1)
		return
	}
	next(l)
}
