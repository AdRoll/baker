package inputtest

import (
	"github.com/AdRoll/baker"
)

// LogLineDesc describes the LogLine input. This input is made for testing.
var LogLineDesc = baker.InputDesc{
	Name:   "LogLine",
	New:    NewLogLine,
	Config: &LogLineConfig{},
}

// LogLineConfig holds the log lines to be fed to the Baker topology.
type LogLineConfig struct {
	Lines []*baker.LogLine
}

// A LogLine input is a Baker input used for testing.
type LogLine struct {
	LogLineConfig
}

// NewLogLine creates a LogLine baker input.
func NewLogLine(cfg baker.InputParams) (baker.Input, error) {
	if cfg.DecodedConfig == nil {
		cfg.DecodedConfig = &LogLineConfig{}
	}
	dcfg := cfg.DecodedConfig.(*LogLineConfig)
	return &LogLine{
		LogLineConfig: *dcfg,
	}, nil
}

func (in *LogLine) Run(output chan<- *baker.Data) error {

	var buf []byte

	// Send all lines via a single baker.Data blob.
	for _, ll := range in.Lines {
		buf = ll.ToText(buf)
		buf = append(buf, '\n')
	}
	output <- &baker.Data{Bytes: buf}

	return nil
}

func (in *LogLine) Stop()                           {}
func (in *LogLine) FreeMem(data *baker.Data)        {}
func (in *LogLine) Stats() (stats baker.InputStats) { return }
