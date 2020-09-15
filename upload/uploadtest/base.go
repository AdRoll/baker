package uploadtest

import "github.com/AdRoll/baker"

// Base is a nop implementation of baker.Upload useful to be embedded in tests
// and to redeclare one or more methods.
type Base struct{}

func (Base) Run(_ <-chan string) error { return nil }
func (Base) Stop()                     {}
func (Base) Stats() baker.UploadStats  { return baker.UploadStats{} }
