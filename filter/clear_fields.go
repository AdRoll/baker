package filter

import (
	"fmt"
	"sort"
	"sync/atomic"

	"github.com/AdRoll/baker"
)

// ClearFieldsDesc describes the ClearFields filter
var ClearFieldsDesc = baker.FilterDesc{
	Name:   "ClearFields",
	New:    NewClearFields,
	Config: &ClearFieldsConfig{},
	Help:   "Clear a set of fields (makes them empty) of all records passing through",
}

// ClearFieldsConfig holds config parameters of the ClearFields filter.
type ClearFieldsConfig struct {
	Fields []string `help:"set of fields to clear" required:"true"`
}

// ClearFields filter clears (i.e set to the empty string) a set of fields.
type ClearFields struct {
	nlines int64
	fields []baker.FieldIndex
}

// NewClearFields returns a ClearFields filter.
func NewClearFields(cfg baker.FilterParams) (baker.Filter, error) {
	if cfg.DecodedConfig == nil {
		cfg.DecodedConfig = &ClearFieldsConfig{}
	}
	dcfg := cfg.DecodedConfig.(*ClearFieldsConfig)

	var fields []baker.FieldIndex

	for _, fname := range dcfg.Fields {
		fidx, ok := cfg.FieldByName(fname)
		if !ok {
			return nil, fmt.Errorf("ClearFields: unknown field %s", fname)
		}
		fields = append(fields, fidx)
	}
	sort.Slice(fields, func(i, j int) bool { return fields[i] < fields[j] })
	return &ClearFields{fields: fields}, nil
}

// Stats returns filter statistics.
func (f *ClearFields) Stats() baker.FilterStats {
	return baker.FilterStats{
		NumProcessedLines: atomic.LoadInt64(&f.nlines),
		NumFilteredLines:  atomic.LoadInt64(&f.nlines),
	}
}

// Process is where the actual filtering takes place.
func (f *ClearFields) Process(l baker.Record, next func(baker.Record)) {
	atomic.AddInt64(&f.nlines, 1)
	for _, fidx := range f.fields {
		l.Set(fidx, nil)
	}
	next(l)
}
