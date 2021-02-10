package filter

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/AdRoll/baker"
)

// shared set of type: map[string]struct{}
var incDedupSet sync.Map

var IncDedupDesc = baker.FilterDesc{
	Name:   "IncDedup",
	New:    NewIncDedup,
	Config: &IncDedupConfig{},
	Help:   "Removes identical records",
}

type IncDedupConfig struct {
	Fields []string `help:"fields that needs to be unique" required:"true"`
}

type IncDedup struct {
	cfg *IncDedupConfig

	fields []baker.FieldIndex

	numProcessedLines int64
	numFilteredLines  int64
}

func NewIncDedup(cfg baker.FilterParams) (baker.Filter, error) {
	if cfg.DecodedConfig == nil {
		cfg.DecodedConfig = &IncDedupConfig{}
	}
	dcfg := cfg.DecodedConfig.(*IncDedupConfig)

	f := &IncDedup{
		cfg: dcfg,
	}
	for _, field := range dcfg.Fields {
		i, ok := cfg.FieldByName(field)
		if !ok {
			return nil, fmt.Errorf("unrecognized deduplication field %q", field)
		}
		f.fields = append(f.fields, i)
	}
	return f, nil
}

func (f *IncDedup) Stats() baker.FilterStats {
	return baker.FilterStats{
		NumProcessedLines: atomic.LoadInt64(&f.numProcessedLines),
		NumFilteredLines:  atomic.LoadInt64(&f.numFilteredLines),
	}
}

func (f *IncDedup) Process(l baker.Record, next func(baker.Record)) {
	atomic.AddInt64(&f.numProcessedLines, 1)

	key := f.constructKey(l)
	_, found := incDedupSet.LoadOrStore(key, struct{}{})

	if found {
		atomic.AddInt64(&f.numFilteredLines, 1)
	} else {
		next(l)
	}
}

// constructKey build a key with the concatenation of all the fields
func (f *IncDedup) constructKey(l baker.Record) string {
	var sb strings.Builder
	for _, i := range f.fields {
		sb.Write(l.Get(i))
	}
	return sb.String()
}
