package filter

import (
	"bytes"
	"fmt"
	"strconv"
	"sync/atomic"
	"unicode"

	"github.com/AdRoll/baker"
)

const expandListhelp = `
This filter extracts values from a list formatted field and writes them into other fields of the same 
record. Each field of the list can be mapped to specific records fields through a TOML table. The elements 
of the list are, by default, separated with the ` + "`;`" + ` character, but it is configurable.

### Example

A possible filter configuration is:

	[[filter]]
	name="ExpandList"
		[filter.config]
		Source = "list_data"
		Separator = ";"
		[filter.config.Fields]
		0 = "field1"
		1 = "field2"
		
In this example, the filter extracts the first and the second element of the list present in the field 
` + "`list_data`" + `of the record. Then, the values of that keys will be written into the field 
` + "`field1`" + ` and ` + "`field2`" + ` of the same record.
`

var ExpandListDesc = baker.FilterDesc{
	Name:   "ExpandList",
	New:    NewExpandList,
	Config: &ExpandListConfig{},
	Help:   expandListhelp,
}

type ExpandListConfig struct {
	Source    string            `help:"record field that contains the list" required:"true"`
	Fields    map[string]string `help:"<list index -> record field> map, the rest will be ignored" required:"true"`
	Separator string            `help:"character separator of the list" required:"false" default:";"`
}

func (c *ExpandListConfig) fillDefault() {
	if c.Separator == "" {
		c.Separator = ";"
	}
}

type ExpandList struct {
	cfg *ExpandListConfig

	source  baker.FieldIndex
	fields  []baker.FieldIndex
	listIdx []int
	sep     []byte

	// Shared state
	numProcessedLines int64
}

func NewExpandList(cfg baker.FilterParams) (baker.Filter, error) {
	dcfg := cfg.DecodedConfig.(*ExpandListConfig)
	dcfg.fillDefault()

	f := &ExpandList{cfg: dcfg}

	idx, ok := cfg.FieldByName(dcfg.Source)
	if !ok {
		return nil, fmt.Errorf("unknown field %q", dcfg.Source)
	}
	f.source = idx

	for k, v := range dcfg.Fields {
		fIdx, ok := cfg.FieldByName(v)
		if !ok {
			return nil, fmt.Errorf("unknown field %q", v)
		}
		f.fields = append(f.fields, fIdx)

		lIdx, err := strconv.Atoi(k)
		if err != nil || lIdx < 0 {
			return nil, fmt.Errorf("invalid integer value %q", k)
		}
		f.listIdx = append(f.listIdx, lIdx)
	}

	sep := []rune(dcfg.Separator)
	if len(sep) != 1 || sep[0] > unicode.MaxASCII {
		return nil, fmt.Errorf("separator must be a 1-byte string or hex char")
	}
	f.sep = []byte(dcfg.Separator)

	return f, nil
}

func (f *ExpandList) Stats() baker.FilterStats {
	return baker.FilterStats{
		NumProcessedLines: atomic.LoadInt64(&f.numProcessedLines),
	}
}

func (f *ExpandList) Process(l baker.Record, next func(baker.Record)) {
	atomic.AddInt64(&f.numProcessedLines, 1)

	list := l.Get(f.source)
	part := bytes.Split(list, f.sep)
	for i, idx := range f.listIdx {
		if idx >= len(part) {
			continue
		}
		b := part[idx]
		l.Set(f.fields[i], b)
	}

	next(l)
}
