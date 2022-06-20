package filter_error_handler

import (
	"fmt"

	"github.com/AdRoll/baker"
)

var ClearFieldsDesc = baker.FilterErrorHandlerDesc{
	Name:   "ClearFields",
	New:    NewClearFields,
	Config: &ClearFieldsConfig{},
	Help:   "Handle errors in a filter by clearing (i.e emptying) one or multiple fields in the Record where the error happened.",
}

type ClearFieldsConfig struct {
	Fields []string `help:"Fields to clear in case of error in the filter"`
}

type ClearFields struct {
	fidxs []baker.FieldIndex
}

func NewClearFields(cfg baker.FilterErrorHandlerParams) (baker.FilterErrorHandler, error) {
	dcfg := cfg.DecodedConfig.(*ClearFieldsConfig)

	var fidxs []baker.FieldIndex
	for _, field := range dcfg.Fields {
		fidx, ok := cfg.FieldByName(field)
		if !ok {
			return nil, fmt.Errorf("unknown field name = %q", field)
		}
		fidxs = append(fidxs, fidx)
	}

	return &ClearFields{fidxs: fidxs}, nil
}

func (h *ClearFields) HandleError(_ string, rec baker.Record, _ error) {
	for _, fidx := range h.fidxs {
		rec.Set(fidx, nil)
	}
}
