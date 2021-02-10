package filter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sync/atomic"

	"github.com/AdRoll/baker"
)

var ExpandJSONDesc = baker.FilterDesc{
	Name:   "ExpandJSON",
	New:    NewExpandJSON,
	Config: &ExpandJSONConfig{},
	Help:   "Explodes json objects to other fields.",
}

type ExpandJSONConfig struct {
	Fields          map[string]string `help:"<json field -> record field> map, the rest will be ignored"`
	Source          string            `help:"record field that contains the json"`
	TrueFalseValues []string          `help:"bind the json boolean values to correstponding strings" default:"[\"true\", \"false\"]"`
}

func (cfg *ExpandJSONConfig) fillDefaults() {
	if len(cfg.TrueFalseValues) == 0 {
		cfg.TrueFalseValues = []string{"true", "false"}
	}
}

const trueIdx, falseIdx = 0, 1

type ExpandJSON struct {
	cfg               *ExpandJSONConfig
	Fields            map[string]baker.FieldIndex
	Source            baker.FieldIndex
	TrueFalseValues   [2][]byte
	numProcessedLines int64
}

func NewExpandJSON(cfg baker.FilterParams) (baker.Filter, error) {
	if cfg.DecodedConfig == nil {
		cfg.DecodedConfig = &ExpandJSONConfig{}
	}
	dcfg := cfg.DecodedConfig.(*ExpandJSONConfig)
	dcfg.fillDefaults()
	ut := &ExpandJSON{cfg: dcfg}
	// Source
	val, ok := cfg.FieldByName(dcfg.Source)
	if !ok {
		return nil, fmt.Errorf("field %s unknown, can't expand it", dcfg.Source)
	}
	ut.Source = val
	// Fields
	ut.Fields = make(map[string]baker.FieldIndex)
	for k, v := range dcfg.Fields {
		val, ok := cfg.FieldByName(v)
		if !ok {
			return nil, fmt.Errorf("field %s unknown, can't expand %s into it", v, k)
		}
		ut.Fields[k] = val
	}
	// TrueFalseValues
	if l := len(dcfg.TrueFalseValues); l != 2 {
		return nil, fmt.Errorf("only two True False values allowed, %v given", l)
	}
	ut.TrueFalseValues = [2][]byte{
		[]byte(dcfg.TrueFalseValues[trueIdx]),
		[]byte(dcfg.TrueFalseValues[falseIdx]),
	}

	return ut, nil
}

func (f *ExpandJSON) Stats() baker.FilterStats {
	return baker.FilterStats{
		NumProcessedLines: atomic.LoadInt64(&f.numProcessedLines),
	}
}

func (f *ExpandJSON) Process(l baker.Record, next func(baker.Record)) {
	// custom json decoder to get all json value types as strings
	// really we just want string -> bytes map
	gm := f.processJSON(l.Get(f.Source))

	if gm != nil {
		for k, i := range f.Fields {
			v, ok := gm[k]
			if !ok {
				continue
			}
			l.Set(i, v)
		}
	}
	atomic.AddInt64(&f.numProcessedLines, 1)
	next(l)
}

func (f *ExpandJSON) processJSON(data []byte) map[string][]byte {
	if len(data) == 0 {
		return nil
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	var x map[string]interface{}
	if err := d.Decode(&x); err != nil {
		return nil
	}
	gm := make(map[string][]byte)
	for k, v := range x {
		switch typedValue := v.(type) {
		case json.Number:
			gm[k] = []byte(typedValue)
		case string:
			gm[k] = []byte(typedValue)
		case bool:
			if typedValue {
				gm[k] = f.TrueFalseValues[trueIdx]
			} else {
				gm[k] = f.TrueFalseValues[falseIdx]
			}
		default:
			// skip other values, including nested json
		}
	}
	return gm
}
