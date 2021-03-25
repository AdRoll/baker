package filter

import (
	"errors"
	"fmt"

	"github.com/AdRoll/baker"
)

// ReplaceFieldsDesc describes the ReplaceFields filter
var ReplaceFieldsDesc = baker.FilterDesc{
	Name:   "ReplaceFields",
	New:    NewReplaceFields,
	Config: &ReplaceFieldsConfig{},
	Help:   `Copy a field value or a fixed value to another field. Can copy multiple fields.`,
}

type ReplaceFieldsConfig struct {
	CopyFields    []string `help:"List of src, dst field pairs, for example [\"srcField1\", \"dstField1\", \"srcField2\", \"dstField2\"]"`
	ReplaceFields []string `help:"List of field, value pairs, for example: [\"Foo\", \"dstField1\", \"Bar\", \"dstField2\"]"`
}

type ReplaceFields struct {
	copyFields       [][2]baker.FieldIndex
	replaceFieldsSrc [][]byte
	replaceFieldsDst []baker.FieldIndex
}

func NewReplaceFields(cfg baker.FilterParams) (baker.Filter, error) {
	dcfg := cfg.DecodedConfig.(*ReplaceFieldsConfig)

	if len(dcfg.CopyFields) == 0 && len(dcfg.ReplaceFields) == 0 {
		return nil, errors.New("either CopyFields or ReplaceFields must be configured")
	}

	if len(dcfg.CopyFields)%2 != 0 {
		return nil, errors.New("CopyFields must contain an even number of fields, in the form [\"sourceField1\", \"destField1\", \"sourceField2\", \"destField2\"]")
	}

	if len(dcfg.ReplaceFields)%2 != 0 {
		return nil, errors.New("ReplaceFields must contain an even number of fields, in the form [\"fixedValue1\", \"destField1\", \"fixedValue2\", \"destField2\"]")
	}

	replaceFieldsSrc := [][]byte{}
	replaceDstMap := map[string]struct{}{}
	replaceFieldsDst := []baker.FieldIndex{}
	for i := 0; i < len(dcfg.ReplaceFields); i = i + 2 {
		replaceFieldsSrc = append(replaceFieldsSrc, []byte(dcfg.ReplaceFields[i]))

		dst, ok := cfg.FieldByName(dcfg.ReplaceFields[i+1])
		if !ok {
			return nil, fmt.Errorf("can't find field %s", dcfg.ReplaceFields[i+1])
		}

		if _, ok := replaceDstMap[dcfg.ReplaceFields[i+1]]; ok {
			return nil, fmt.Errorf("field %s used multiple times for replacements", dcfg.ReplaceFields[i+1])
		}
		replaceDstMap[dcfg.ReplaceFields[i+1]] = struct{}{}

		replaceFieldsDst = append(replaceFieldsDst, dst)
	}

	copyFields := [][2]baker.FieldIndex{}
	copyDstMap := map[string]struct{}{}
	for i := 0; i < len(dcfg.CopyFields); i = i + 2 {
		src, ok := cfg.FieldByName(dcfg.CopyFields[i])
		if !ok {
			return nil, fmt.Errorf("can't find field %s", dcfg.CopyFields[i])
		}

		dst, ok := cfg.FieldByName(dcfg.CopyFields[i+1])
		if !ok {
			return nil, fmt.Errorf("can't find field %s", dcfg.CopyFields[i+1])
		}

		if src == dst {
			return nil, fmt.Errorf("wrong config, replacing the same field: %s", dcfg.CopyFields[i])
		}

		if _, ok := copyDstMap[dcfg.CopyFields[i+1]]; ok {
			return nil, fmt.Errorf("field %s used multiple times as copy destination", dcfg.CopyFields[i+1])
		}

		if _, ok := replaceDstMap[dcfg.CopyFields[i+1]]; ok {
			return nil, fmt.Errorf("field %s used both as copy and replacement destination", dcfg.CopyFields[i+1])
		}
		copyDstMap[dcfg.CopyFields[i+1]] = struct{}{}

		copyFields = append(copyFields, [2]baker.FieldIndex{src, dst})
	}

	return &ReplaceFields{
		copyFields:       copyFields,
		replaceFieldsSrc: replaceFieldsSrc,
		replaceFieldsDst: replaceFieldsDst,
	}, nil
}

func (f *ReplaceFields) Process(l baker.Record, next func(baker.Record)) {
	for _, field := range f.copyFields {
		l.Set(field[1], l.Get(field[0]))
	}

	for i, v := range f.replaceFieldsSrc {
		l.Set(f.replaceFieldsDst[i], v)
	}

	next(l)
}

func (c *ReplaceFields) Stats() baker.FilterStats {
	return baker.FilterStats{}
}
