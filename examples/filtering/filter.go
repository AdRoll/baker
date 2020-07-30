package main

import (
	"time"

	"github.com/AdRoll/baker"
)

var LazyFilterDesc = baker.FilterDesc{
	Name:   "LazyFilter",
	New:    NewLazyFilter,
	Config: &LazyFilterConfig{},
	Help:   "This lazy filter does nothing during working time but drops all records between 6pm and 9am",
}

type LazyFilterConfig struct {
	Stakhanovite bool `help:"If true, then only discard records after 8pm"`
}
type LazyFilter struct {
	stakhanovite bool
}

func NewLazyFilter(cfg baker.FilterParams) (baker.Filter, error) {
	if cfg.DecodedConfig == nil {
		cfg.DecodedConfig = &LazyFilterConfig{}
	}
	dcfg := cfg.DecodedConfig.(*LazyFilterConfig)
	return &LazyFilter{
		stakhanovite: dcfg.Stakhanovite,
	}, nil
}

func (f *LazyFilter) Process(l baker.Record, next func(baker.Record)) {
	h := time.Now().Hour()
	upperLimit := 18
	if f.stakhanovite {
		upperLimit = 20
	}
	if h >= 9 && h <= upperLimit {
		next(l)
	}
}

func (f *LazyFilter) Stats() baker.FilterStats {
	return baker.FilterStats{}
}
