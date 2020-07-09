package inputtest

import (
	"github.com/AdRoll/baker"
)

// RecordsDesc describes the Records input. This input is made for testing.
var RecordsDesc = baker.InputDesc{
	Name:   "Records",
	New:    NewRecords,
	Config: &RecordsConfig{},
}

// RecordsConfig holds the log lines to be fed to the Baker topology.
type RecordsConfig struct {
	Records []baker.Record
}

// A Records input is a Baker input used for testing.
type Records struct {
	RecordsConfig
}

// NewRecords creates a Records baker input.
func NewRecords(cfg baker.InputParams) (baker.Input, error) {
	if cfg.DecodedConfig == nil {
		cfg.DecodedConfig = &RecordsConfig{}
	}
	dcfg := cfg.DecodedConfig.(*RecordsConfig)
	return &Records{
		RecordsConfig: *dcfg,
	}, nil
}

func (in *Records) Run(output chan<- *baker.Data) error {

	var buf []byte

	// Send all lines via a single baker.Data blob.
	for _, ll := range in.Records {
		buf = ll.ToText(buf)
		buf = append(buf, '\n')
	}
	output <- &baker.Data{Bytes: buf}

	return nil
}

func (Records) Stop()                           {}
func (Records) FreeMem(data *baker.Data)        {}
func (Records) Stats() (stats baker.InputStats) { return }
