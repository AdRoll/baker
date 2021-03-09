package filter

import (
	"bytes"
	"fmt"
	"sync/atomic"

	"github.com/AdRoll/baker"
)

// StringMatchDesc describes the StringMatch filter
var StringMatchDesc = baker.FilterDesc{
	Name:   "StringMatch",
	New:    NewStringMatch,
	Config: &StringMatchConfig{},
	Help:   "Discard records if a field matches any of the provided strings",
}

// StringMatchConfig holds config parameters of the StringMatch filter.
type StringMatchConfig struct {
	Field       string   `help:"name of the field which value is used for string comparison" required:"true"`
	Strings     []string `help:"list of strings to match." required:"true"`
	InvertMatch bool     `help:"Invert the match outcome, so that records are discarded if they don't match any of the strings" default:"false"`
}

// StringMatch filter clears (i.e set to the empty string) a set of fields.
type StringMatch struct {
	field   baker.FieldIndex
	strings [][]byte
	invert  bool

	processed int64
	discarded int64
}

// NewStringMatch returns a StringMatch filter.
func NewStringMatch(cfg baker.FilterParams) (baker.Filter, error) {
	dcfg := cfg.DecodedConfig.(*StringMatchConfig)

	if len(dcfg.Strings) == 0 {
		return nil, fmt.Errorf("At least one string must be defined in Strings")
	}

	fidx, ok := cfg.FieldByName(dcfg.Field)
	if !ok {
		return nil, fmt.Errorf("StringMatch: unknown field %s", dcfg.Field)
	}

	var strings [][]byte
	for i := range dcfg.Strings {
		strings = append(strings, []byte(dcfg.Strings[i]))
	}

	return &StringMatch{field: fidx, strings: strings, invert: dcfg.InvertMatch}, nil
}

// Stats returns filter statistics.
func (f *StringMatch) Stats() baker.FilterStats {
	return baker.FilterStats{
		NumProcessedLines: atomic.LoadInt64(&f.processed),
		NumFilteredLines:  atomic.LoadInt64(&f.discarded),
	}
}

func (f *StringMatch) isMatchAny(l baker.Record) bool {
	buf := l.Get(f.field)
	for i := range f.strings {
		if bytes.Equal(buf, f.strings[i]) {
			return true
		}
	}

	return false
}

// Process is where the actual filtering takes place.
func (f *StringMatch) Process(l baker.Record, next func(baker.Record)) {
	atomic.AddInt64(&f.processed, 1)

	if f.isMatchAny(l) == !f.invert {
		atomic.AddInt64(&f.discarded, 1)
		return
	}

	next(l)
}
