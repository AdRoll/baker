package outputtest

import "github.com/AdRoll/baker"

var BaseDesc = baker.OutputDesc{
	Name:   "Base",
	New:    NewBase,
	Config: &BaseConfig{},
	Raw:    true,
}

type BaseConfig struct {
	SupportConcurrency bool `help:"Add concurrency support" default:"false"`
}

// Base is a nop implementation of baker.Output useful to be embedded in tests
// and to redeclare one or more methods.
type Base struct {
	supportConcurrency bool
}

func NewBase(cfg baker.OutputParams) (baker.Output, error) {
	return &Base{
		supportConcurrency: cfg.DecodedConfig.(*BaseConfig).SupportConcurrency,
	}, nil
}

func (Base) Run(_ <-chan baker.OutputRecord, _ chan<- string) error { return nil }
func (Base) CanShard() bool                                         { return false }

func (b *Base) SupportConcurrency() bool {
	return b.supportConcurrency
}

func (Base) Stats() baker.OutputStats { return baker.OutputStats{} }
