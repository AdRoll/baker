package filter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sync/atomic"

	"github.com/AdRoll/baker"
	"github.com/jmespath/go-jmespath"
)

var ExpandJSONDesc = baker.FilterDesc{
	Name:   "ExpandJSON",
	New:    NewExpandJSON,
	Config: &ExpandJSONConfig{},
	Help:   "Explodes json objects to other fields.",
}

type ExpandJSONConfig struct {
	Fields          map[string]string `help:"<JMESPath -> record field> map, the rest will be ignored"`
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
	cfg *ExpandJSONConfig

	fields          []baker.FieldIndex
	jexp            []*jmespath.JMESPath
	source          baker.FieldIndex
	trueFalseValues [2][]byte

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
	ut.source = val
	// Fields
	for j, f := range dcfg.Fields {
		i, ok := cfg.FieldByName(f)
		if !ok {
			return nil, fmt.Errorf("field %q unknown, can't expand %q into it", f, j)
		}
		c, err := jmespath.Compile(j)
		if err != nil {
			return nil, fmt.Errorf("can't compile JMESPath expression %q for field %q", j, f)
		}
		ut.fields = append(ut.fields, i)
		ut.jexp = append(ut.jexp, c)
	}
	// TrueFalseValues
	if l := len(dcfg.TrueFalseValues); l != 2 {
		return nil, fmt.Errorf("only two True False values allowed, %v given", l)
	}
	ut.trueFalseValues = [2][]byte{
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
	data := f.processJSON(l.Get(f.source))

	for i, c := range f.jexp {
		r, err := c.Search(data)
		if err != nil || r == nil {
			continue
		}
		l.Set(f.fields[i], f.postProcessJSON(r))
	}

	atomic.AddInt64(&f.numProcessedLines, 1)
	next(l)
}

func (f *ExpandJSON) processJSON(data []byte) interface{} {
	if len(data) == 0 {
		return nil
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber() // leave numbers as strings
	var x interface{}
	if err := d.Decode(&x); err != nil {
		return nil
	}
	return x
}

func (f *ExpandJSON) postProcessJSON(r interface{}) (val []byte) {
	switch typedValue := r.(type) {
	case json.Number:
		val = []byte(typedValue)
	case string:
		val = []byte(typedValue)
	case bool:
		if typedValue {
			val = f.trueFalseValues[trueIdx]
		} else {
			val = f.trueFalseValues[falseIdx]
		}
	default:
		val, _ = json.Marshal(typedValue)
	}
	return
}
