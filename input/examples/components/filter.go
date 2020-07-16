// This is an example of filter, including its toml configuration.
// The filter gets a record's field name and a possible value from configuration.
// All records with different values for that field are filtered out.
//
// [[filter]]
// name = "MyFilter"
//     [filter.config]
//     FieldName = "Company"
//    AcceptedValue = "NextRoll"
package main

import (
	"bytes"
	"errors"
	"sync/atomic"

	"github.com/AdRoll/baker"
)

var MyFilterDesc = baker.FilterDesc{
	Name:   "MyFilter",
	New:    NewMyFilter,
	Config: &MyFilterConfig{},
	Help:   `Drops lines with invalid value for the given field`,
}

type MyFilterConfig struct {
	FieldName     string `help:"The name of the field to filter on"`
	AcceptedValue string `help:"The accepted value for the filtered field"`
}

type MyFilter struct {
	numProcessedLines int64
	numFilteredLines  int64
	cfg               *MyFilterConfig
	idx               baker.FieldIndex
}

func NewMyFilter(cfg baker.FilterParams) (baker.Filter, error) {
	if cfg.DecodedConfig == nil {
		cfg.DecodedConfig = &MyFilterConfig{}
	}
	dcfg := cfg.DecodedConfig.(*MyFilterConfig)
	idx, ok := cfg.FieldByName(dcfg.FieldName)
	if !ok {
		return nil, errors.New("Some error")
	}
	return &MyFilter{cfg: dcfg, idx: idx}, nil
}

func (f *MyFilter) Stats() baker.FilterStats {
	return baker.FilterStats{
		NumProcessedLines: atomic.LoadInt64(&f.numProcessedLines),
		NumFilteredLines:  atomic.LoadInt64(&f.numFilteredLines),
	}
}

func (f *MyFilter) Process(l baker.Record, next func(baker.Record)) {
	atomic.AddInt64(&f.numProcessedLines, 1)

	if !bytes.Equal(l.Get(f.idx), []byte(f.cfg.AcceptedValue)) {
		atomic.AddInt64(&f.numFilteredLines, 1)
		// Filter out the record not calling next()
		return
	}
	// Call next filter in the filter chain
	next(l)
}
