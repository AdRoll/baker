package output

import (
	"sync/atomic"

	"github.com/AdRoll/baker"
	log "github.com/sirupsen/logrus"
)

var OpLogDesc = baker.OutputDesc{
	Name:   "OpLog",
	New:    NewOpLog,
	Config: &OpLogConfig{},
	Raw:    false,
	Help:   "This output writes the filtered log lines into the current baker log, purely for development purpose.\n",
}

type OpLogConfig struct{}

func (cfg *OpLogConfig) fillDefaults() {}

type OpLog struct {
	Cfg *OpLogConfig

	Fields []baker.FieldIndex
	totaln int64
}

func NewOpLog(cfg baker.OutputParams) (baker.Output, error) {
	log.WithFields(log.Fields{"fn": "NewOpLog", "idx": cfg.Index}).Info("Initializing")

	if cfg.DecodedConfig == nil {
		cfg.DecodedConfig = &OpLogConfig{}
	}
	dcfg := cfg.DecodedConfig.(*OpLogConfig)
	dcfg.fillDefaults()

	return &OpLog{
		Cfg:    dcfg,
		Fields: cfg.Fields,
	}, nil
}

func (w *OpLog) Run(input <-chan baker.OutputRecord, _ chan<- string) {
	log.Info("OpLog ready to log")
	for lldata := range input {
		log.WithFields(log.Fields{"line": lldata.Fields}).Info(".")
		atomic.AddInt64(&w.totaln, int64(1))
	}
}

func (w *OpLog) Stats() baker.OutputStats {
	return baker.OutputStats{
		NumProcessedLines: atomic.LoadInt64(&w.totaln),
	}
}

func (b *OpLog) CanShard() bool {
	return false
}
