package filter

import (
	"fmt"
	"sync/atomic"

	"github.com/AdRoll/baker"
	"github.com/SemanticSugar/baker/forklift"
)

// TODO[open-source] this can become a generic filter that filters out all lines with a null value for
// the given field, but the configuration must work with field idx

var NotNullDesc = baker.FilterDesc{
	Name:   "NotNull",
	New:    NewNotNull,
	Config: &NotNullConfig{},
	Help:   "Discard lines with null fields.\n",
}

type NotNullConfig struct {
	Fields []string `help:"Which fields to check for null values." default:"[\"advertisable_eid\"]"`
}

type NotNull struct {
	numProcessedLines int64
	numFilteredLines  int64
	cfg               *NotNullConfig
	fields            []baker.FieldIndex
}

func (cfg *NotNullConfig) fillDefaults() {
	if len(cfg.Fields) == 0 {
		cfg.Fields = []string{"advertisable_eid"}
	}
}

func NewNotNull(cfg baker.FilterParams) (baker.Filter, error) {
	if cfg.DecodedConfig == nil {
		cfg.DecodedConfig = &NotNullConfig{}
	}
	dcfg := cfg.DecodedConfig.(*NotNullConfig)
	dcfg.fillDefaults()

	f := &NotNull{cfg: dcfg}
	for _, field := range dcfg.Fields {
		if val, ok := forklift.FieldByName(field); ok {
			f.fields = append(f.fields, val)
		} else {
			return nil, fmt.Errorf("cannot find field '%s' to check for null values", field)
		}
	}
	return f, nil
}

func (v *NotNull) Stats() baker.FilterStats {
	return baker.FilterStats{
		NumProcessedLines: atomic.LoadInt64(&v.numProcessedLines),
		NumFilteredLines:  atomic.LoadInt64(&v.numFilteredLines),
	}
}

func (v *NotNull) Process(l baker.Record, next func(baker.Record)) {
	atomic.AddInt64(&v.numProcessedLines, 1)
	for _, field := range v.fields {
		if l.Get(field) == nil {
			atomic.AddInt64(&v.numFilteredLines, 1)
			return
		}
	}
	next(l)
}
