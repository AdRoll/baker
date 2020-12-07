package output

import (
	"sync/atomic"

	"github.com/AdRoll/baker"
)

var NopDesc = baker.OutputDesc{
	Name:   "Nop",
	New:    NewNop,
	Config: &NopConfig{},
	Help:   "No-operation output. This output simply drops all lines and does not write them anywhere.",
}

type Nop struct{ totaln int64 }

type NopConfig struct{}

func NewNop(cfg baker.OutputParams) (baker.Output, error) {
	return &Nop{}, nil
}

func (b *Nop) CanShard() bool { return true }

func (nop *Nop) Run(input <-chan baker.OutputRecord, upch chan<- string) error {
	for range input {
		atomic.AddInt64(&nop.totaln, 1)
	}

	return nil
}

func (nop *Nop) Stats() baker.OutputStats {
	return baker.OutputStats{
		NumProcessedLines: atomic.LoadInt64(&nop.totaln),
		NumErrorLines:     0,
	}
}
