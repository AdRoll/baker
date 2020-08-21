package inputtest

import "github.com/AdRoll/baker"

// Base is a nop implementation of baker.Input useful to be embedded in tests
// and to redeclare one or more methods.
type Base struct{}

func (Base) Run(_ chan<- *baker.Data) error { return nil }
func (Base) Stop()                          {}
func (Base) FreeMem(*baker.Data)            {}
func (Base) Stats() baker.InputStats        { return baker.InputStats{} }
