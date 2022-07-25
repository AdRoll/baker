package filter

import (
	"fmt"

	"github.com/AdRoll/baker"
)

// NotNullDesc describes the NotNull filter.
var NotNullDesc = baker.FilterDesc{
	Name:   "NotNull",
	New:    NewNotNull,
	Config: &NotNullConfig{},
	Help:   "Generates errors on records having null (i.e empty) fields.\n",

	DropOnErrorDefault: true,
}

// NotNullConfig holds configuration parameters for the NotNull filter.
type NotNullConfig struct {
	Fields []string `help:"Fields is the list of fields to check for null/empty values" required:"true"`
}

// NotNull is a baker filter that discards records having null fields.
type NotNull struct {
	cfg    *NotNullConfig
	fields []baker.FieldIndex
}

// NewNotNull creates and configures a new NotNull filter.
func NewNotNull(cfg baker.FilterParams) (baker.Filter, error) {
	dcfg := cfg.DecodedConfig.(*NotNullConfig)

	f := &NotNull{cfg: dcfg}
	for _, field := range dcfg.Fields {
		if val, ok := cfg.FieldByName(field); ok {
			f.fields = append(f.fields, val)
		} else {
			return nil, fmt.Errorf("unknown field %q", field)
		}
	}
	return f, nil
}

// Stats implements baker.Filter.
func (v *NotNull) Stats() baker.FilterStats {
	return baker.FilterStats{}
}

// Process implements baker.Filter.
func (v *NotNull) Process(l baker.Record) error {
	for _, field := range v.fields {
		if l.Get(field) == nil {
			return baker.ErrGenericFilterError
		}
	}
	return nil
}
