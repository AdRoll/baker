package filtertest

import (
	"github.com/AdRoll/baker"
)

// PassThroughDesc describes the PassThrough filter.
var PassThroughDesc = baker.FilterDesc{
	Name:   "PassThrough",
	New:    newPassThrough,
	Config: &passThrough{},
	Help:   "let all through, useful for test/debug purposes",
}

type passThrough struct{}

func newPassThrough(icfg baker.FilterParams) (baker.Filter, error) { return &passThrough{}, nil }
func (f *passThrough) Stats() baker.FilterStats                    { return baker.FilterStats{} }
func (f *passThrough) Process(l baker.Record) error                { return nil }
