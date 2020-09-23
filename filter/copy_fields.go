package filter

import (
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/AdRoll/baker"
)

var CopyFieldsDesc = baker.FilterDesc{
	Name:   "CopyFields",
	New:    NewCopyFields,
	Config: &CopyFieldsConfig{},
	Help:   `Copy a field value to another field. Can copy multiple fields.`,
}

type CopyFieldsConfig struct {
	FieldsMap []string `help:"List of fields to replace, as: [\"sourceField1\", \"targetField1\", \"sourceField2\", \"targetField2\"]" required:"true"`
}

type CopyFields struct {
	numProcessedLines int64
	fieldsMap         [][2]baker.FieldIndex
}

func NewCopyFields(cfg baker.FilterParams) (baker.Filter, error) {
	if cfg.DecodedConfig == nil {
		cfg.DecodedConfig = &CopyFieldsConfig{}
	}
	dcfg := cfg.DecodedConfig.(*CopyFieldsConfig)

	if len(dcfg.FieldsMap)%2 != 0 {
		return nil, errors.New("FieldsMap must contain an even number of fields, in the form [\"sourceField1\", \"targetField1\", \"sourceField2\", \"targetField2\"]")
	}

	fieldsMap := [][2]baker.FieldIndex{}
	for i := 0; i < len(dcfg.FieldsMap); i = i + 2 {
		src, ok := cfg.FieldByName(dcfg.FieldsMap[i])
		if !ok {
			return nil, fmt.Errorf("Can't find field %s", dcfg.FieldsMap[i])
		}

		trgt, ok := cfg.FieldByName(dcfg.FieldsMap[i+1])
		if !ok {
			return nil, fmt.Errorf("Can't find field %s", dcfg.FieldsMap[i+1])
		}

		if src == trgt {
			return nil, fmt.Errorf("Wrong config, replacing the same field: %s", dcfg.FieldsMap[i])
		}

		fieldsMap = append(fieldsMap, [2]baker.FieldIndex{src, trgt})
	}

	return &CopyFields{
		fieldsMap: fieldsMap,
	}, nil
}

func (f *CopyFields) Process(l baker.Record, next func(baker.Record)) {
	atomic.AddInt64(&f.numProcessedLines, 1)

	for _, field := range f.fieldsMap {
		l.Set(field[1], l.Get(field[0]))
	}

	next(l)
}

func (c *CopyFields) Stats() baker.FilterStats {
	return baker.FilterStats{
		NumProcessedLines: atomic.LoadInt64(&c.numProcessedLines),
	}
}
