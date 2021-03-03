package filter

import (
	"bytes"
	"fmt"
	"sync"
	"sync/atomic"
	"unicode"

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
	Fields       []string `help:"fields to consider when comparing records" required:"true"`
	KeySeparator string   `help:"character separator used to build a key from the fields" default:"\\x1e"`
}

func (cfg *DedupConfig) fillDefaults() {
	if cfg.KeySeparator == "" {
		cfg.KeySeparator = "\x1e"
	}
}

type Dedup struct {
	cfg *DedupConfig

	fields []baker.FieldIndex
	sep    []byte

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
	dcfg.fillDefaults()

	f := &Dedup{cfg: dcfg}

	for _, field := range dcfg.Fields {
		i, ok := cfg.FieldByName(field)
		if !ok {
			return nil, fmt.Errorf("unknown field %q", field)
		}
		f.fields = append(f.fields, i)
	}

	sep := []rune(dcfg.KeySeparator)
	if len(sep) != 1 || sep[0] > unicode.MaxASCII {
		return nil, fmt.Errorf("separator must be a 1-byte string or hex char")
	}
	f.sep = []byte(dcfg.KeySeparator)

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
	if _, found := f.dedupSet.LoadOrStore(key, struct{}{}); found {
		atomic.AddInt64(&f.numFilteredLines, 1)
		return
	}

	next(l)
}

// constructKey builds a key by concatenating field values
func (f *Dedup) constructKey(l baker.Record) string {
	fields := make([][]byte, len(f.fields))
	for i, idx := range f.fields {
		fields[i] = l.Get(idx)
	}
	return string(bytes.Join(fields, f.sep))
}
