package filter

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/AdRoll/baker"
)

const help = `
Removes record with identical value using a set of provided fields as key.

The deduplication procedure internally uses a shared set, concurrently used between all the filter replicas for storing the encountered record key.
Indeed, the use of this filter could introduce performance degradation and possible unbounded memory consumption.

The user is responsible to properly chouse the set of fields to construct a bounded combination of possible inputs and thus maintaining the
memory consumption under control.
`

var IncDedupDesc = baker.FilterDesc{
	Name:   "IncDedup",
	New:    NewIncDedup,
	Config: &IncDedupConfig{},
	Help:   help,
}

type IncDedupConfig struct {
	Fields []string `help:"fields that needs to be unique" required:"true"`
}

type IncDedup struct {
	cfg *IncDedupConfig

	fields []baker.FieldIndex

	// Shared state
	dedupSet          sync.Map // type: map[string]struct{}
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
	_, found := f.dedupSet.LoadOrStore(key, struct{}{})

	if found {
		atomic.AddInt64(&f.numFilteredLines, 1)
	} else {
		next(l)
	}
}

// constructKey build a key with the concatenation of the fields
func (f *IncDedup) constructKey(l baker.Record) string {
	var sb strings.Builder
	for _, i := range f.fields {
		sb.Write(l.Get(i))
	}
	return sb.String()
}
