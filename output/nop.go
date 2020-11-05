package output

import (
	"sync/atomic"

	"github.com/AdRoll/baker"
)

var NopDesc = baker.OutputDesc{
	Name:   "Nop",
	New:    NewNopWriter,
	Config: &NopWriterConfig{},
	Help:   "No-operation output. This output simply drops all lines and does not write them anywhere.",
}

type NopWriter struct{ totaln int64 }

type NopWriterConfig struct{}

func NewNopWriter(cfg baker.OutputParams) (baker.Output, error) {
	return &NopWriter{}, nil
}

func (b *NopWriter) CanShard() bool           { return true }
func (b *NopWriter) SupportConcurrency() bool { return true }

func (nop *NopWriter) Run(input <-chan baker.OutputRecord, upch chan<- string) error {
	for range input {
		atomic.AddInt64(&nop.totaln, 1)
	}

	return nil
}

func (nop *NopWriter) Stats() baker.OutputStats {
	return baker.OutputStats{
		NumProcessedLines: atomic.LoadInt64(&nop.totaln),
		NumErrorLines:     0,
	}
}
