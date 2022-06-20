package filter_error_handler

import (
	"fmt"

	"github.com/AdRoll/baker"

	log "github.com/sirupsen/logrus"
)

var LogDesc = baker.FilterErrorHandlerDesc{
	Name:   "Log",
	New:    NewLog,
	Config: &LogConfig{},
	Help:   "Handle errors by logging them, alongside a selection of fields, to standard output or standard error.",
}

type LogConfig struct {
	Fields []string `help:"Fields which values should be added to the log message."`
	Level  string   `help:"Level of the log messages. Accepts 'panic', 'fatal', 'error', 'warn', 'info', 'debug' or 'trace'" default:"error"`
}

type Log struct {
	fidxs  []baker.FieldIndex
	fnames []string
	lvl    log.Level
}

func NewLog(cfg baker.FilterErrorHandlerParams) (baker.FilterErrorHandler, error) {
	dcfg := cfg.DecodedConfig.(*LogConfig)

	var (
		fidxs  []baker.FieldIndex
		fnames []string
	)
	for _, field := range dcfg.Fields {
		fidx, ok := cfg.FieldByName(field)
		if !ok {
			return nil, fmt.Errorf("unknown field name = %q", field)
		}
		fidxs = append(fidxs, fidx)
		fnames = append(fnames, field)
	}

	if dcfg.Level == "" {
		dcfg.Level = "error"
	}

	lvl, err := log.ParseLevel(dcfg.Level)
	if err != nil {
		return nil, fmt.Errorf("unexpected log level %v", dcfg.Level)
	}

	h := &Log{
		fidxs:  fidxs,
		fnames: fnames,
		lvl:    lvl,
	}
	return h, nil
}

func (h *Log) HandleError(filterName string, rec baker.Record, err error) {
	logFields := log.Fields{
		"filter_name": filterName,
	}
	for idx, fidx := range h.fidxs {
		logFields[h.fnames[idx]] = string(rec.Get(fidx))
	}
	log.WithFields(logFields).Logf(h.lvl, "dropped record")
}
