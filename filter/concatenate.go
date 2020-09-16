package filter

import (
	"errors"
	"fmt"
	"sync/atomic"
	"unicode"

	"github.com/AdRoll/baker"
)

var ConcatenateDesc = baker.FilterDesc{
	Name:   "Concatenate",
	New:    NewConcatenate,
	Config: &ConcatenateConfig{},
	Help:   `Concatenate up to 10 fields' values to a single field`,
}

type ConcatenateConfig struct {
	Fields    []string `help:"The field names to concatenate, in order"`
	Target    string   `help:"The field name to save the concatenated value to"`
	Separator string   `help:"Use this separator to concatenate the values. Must be a single-byte convertible char"`
}

type Concatenate struct {
	numProcessedLines int64
	numFilteredLines  int64
	fields            []baker.FieldIndex
	target            baker.FieldIndex
	separator         byte
}

func NewConcatenate(cfg baker.FilterParams) (baker.Filter, error) {
	if cfg.DecodedConfig == nil {
		cfg.DecodedConfig = &ConcatenateConfig{}
	}
	dcfg := cfg.DecodedConfig.(*ConcatenateConfig)

	fields := []baker.FieldIndex{}
	for _, fieldName := range dcfg.Fields {
		i, ok := cfg.FieldByName(fieldName)
		if !ok {
			return nil, fmt.Errorf("Can't resolve field %s", fieldName)
		}
		fields = append(fields, i)
	}

	target, ok := cfg.FieldByName(dcfg.Target)
	if !ok {
		return nil, fmt.Errorf("Can't resolve target field %s", dcfg.Target)
	}

	var separator byte
	if dcfg.Separator != "" {
		sep := []rune(dcfg.Separator)
		if len(sep) != 1 || sep[0] > unicode.MaxASCII {
			return nil, errors.New("Separator must be a 1-byte string or hex char")
		}
		separator = byte(sep[0])
	}

	f := &Concatenate{
		fields:    fields,
		target:    target,
		separator: separator,
	}

	return f, nil
}

func (c *Concatenate) Stats() baker.FilterStats {
	return baker.FilterStats{
		NumProcessedLines: atomic.LoadInt64(&c.numProcessedLines),
		NumFilteredLines:  atomic.LoadInt64(&c.numFilteredLines),
	}
}

func (c *Concatenate) Process(l baker.Record, next func(baker.Record)) {
	atomic.AddInt64(&c.numProcessedLines, 1)

	key := make([]byte, 0, 512)
	flen := len(c.fields) - 1
	for i, f := range c.fields {
		v := l.Get(f)
		if i < flen && c.separator != 0 {
			v = append(v, c.separator)
		}
		key = append(key, v...)
	}

	l.Set(c.target, key)
	next(l)
}
