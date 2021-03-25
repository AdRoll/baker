package filter

import (
	"fmt"
	"regexp"
	"sync/atomic"

	"github.com/AdRoll/baker"
)

// RegexMatchDesc describes the RegexMatch filter
var RegexMatchDesc = baker.FilterDesc{
	Name:   "RegexMatch",
	New:    NewRegexMatch,
	Config: &RegexMatchConfig{},
	Help:   "Discard a record if one or more fields don't match the corresponding regular expressions",
}

// RegexMatchConfig holds config parameters of the RegexMatch filter.
type RegexMatchConfig struct {
	Fields []string `help:"list of fields to match with the corresponding regular expression in Regexs" default:"[]"`
	Regexs []string `help:"list of regular expression to match. Fields[0] must match Regexs[0], Fields[1] Regexs[1] and so on" default:"[]"`
}

// RegexMatch filter clears (i.e set to the empty string) a set of fields.
type RegexMatch struct {
	discarded int64

	fields []baker.FieldIndex
	res    []*regexp.Regexp
}

// NewRegexMatch returns a RegexMatch filter.
func NewRegexMatch(cfg baker.FilterParams) (baker.Filter, error) {
	dcfg := cfg.DecodedConfig.(*RegexMatchConfig)

	if len(dcfg.Fields) != len(dcfg.Regexs) {
		return nil, fmt.Errorf("the number of Fields and Regexs should be the same")
	}

	var (
		fields []baker.FieldIndex
		res    []*regexp.Regexp
	)

	for i, fname := range dcfg.Fields {
		fidx, ok := cfg.FieldByName(fname)
		if !ok {
			return nil, fmt.Errorf("RegexMatch: unknown field %s", fname)
		}
		fields = append(fields, fidx)
		re, err := regexp.Compile(dcfg.Regexs[i])
		if err != nil {
			return nil, fmt.Errorf("RegexMatch: Regexs[%d]: %s", i, err)
		}
		res = append(res, re)
	}
	return &RegexMatch{fields: fields, res: res}, nil
}

// Stats returns filter statistics.
func (f *RegexMatch) Stats() baker.FilterStats {
	return baker.FilterStats{
		NumFilteredLines: atomic.LoadInt64(&f.discarded),
	}
}

func (f *RegexMatch) match(l baker.Record) bool {
	for i := range f.fields {
		if !f.res[i].Match(l.Get(f.fields[i])) {
			return false
		}
	}

	return true
}

// Process is where the actual filtering takes place.
func (f *RegexMatch) Process(l baker.Record, next func(baker.Record)) {
	if !f.match(l) {
		atomic.AddInt64(&f.discarded, 1)
		return
	}

	next(l)
}
