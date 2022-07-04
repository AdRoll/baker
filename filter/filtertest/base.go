package filtertest

import "github.com/AdRoll/baker"

// Base is a nop implementation of baker.Filter useful to be embedded in tests
// and to redeclare one or more methods.
type Base struct{}

func (Base) Process(l baker.Record) error { return nil }
func (Base) Stats() baker.FilterStats     { return baker.FilterStats{} }
