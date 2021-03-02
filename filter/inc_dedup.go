package filter

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/AdRoll/baker"
)

const dedupHelp = `
This filter removes duplicate records. A record is considered a duplicate, and is thus removed by this filter, 
if another record with the same values has already been _seen_. The comparison is performed on a 
user-provided list of fields (` + "`Fields`" + ` setting).

**WARNING**: to remove duplicates, this filter stores one key per unique record in memory, this means 
that the overall memory grows linearly with the number of unique records in your data set. Depending 
on your data set, this might lead to OOM (i.e. out of memory) errors.
`

var DedupDesc = baker.FilterDesc{
	Name:   "Dedup",
	New:    NewDedup,
	Config: &DedupConfig{},
	Help:   dedupHelp,
}

type DedupConfig struct {
	Fields []string `help:"fields to consider when comparing records" required:"true"`
}

type Dedup struct {
	cfg *DedupConfig

	fields []baker.FieldIndex

	// Shared state
	dedupSet          sync.Map // type: map[string]struct{}
	numProcessedLines int64
	numFilteredLines  int64
}

func NewDedup(cfg baker.FilterParams) (baker.Filter, error) {
	if cfg.DecodedConfig == nil {
		cfg.DecodedConfig = &DedupConfig{}
	}
	dcfg := cfg.DecodedConfig.(*DedupConfig)

	f := &Dedup{
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

func (f *Dedup) Stats() baker.FilterStats {
	return baker.FilterStats{
		NumProcessedLines: atomic.LoadInt64(&f.numProcessedLines),
		NumFilteredLines:  atomic.LoadInt64(&f.numFilteredLines),
	}
}

func (f *Dedup) Process(l baker.Record, next func(baker.Record)) {
	atomic.AddInt64(&f.numProcessedLines, 1)

	key := f.constructKey(l)
	_, found := f.dedupSet.LoadOrStore(key, struct{}{})

	if found {
		atomic.AddInt64(&f.numFilteredLines, 1)
	} else {
		next(l)
	}
}

// constructKey builds a key by concatenating field values
func (f *Dedup) constructKey(l baker.Record) string {
	var sb strings.Builder
	for _, i := range f.fields {
		sb.Write(l.Get(i))
	}
	return sb.String()
}
