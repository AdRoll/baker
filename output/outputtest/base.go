package outputtest

import "github.com/AdRoll/baker"

// Base is a nop implementation of baker.Output useful to be embedded in tests
// and to redeclare one or more methods.
type Base struct{}

func (Base) Run(_ <-chan baker.OutputRecord, _ chan<- string) error { return nil }
func (Base) CanShard() bool                                         { return false }
func (Base) Stats() baker.OutputStats                               { return baker.OutputStats{} }
