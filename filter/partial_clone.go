package filter

import (
	"bytes"
	"fmt"
	"sync/atomic"

	"github.com/AdRoll/baker"
)

const subRecordHelp = `
If the record matches the expected values, the filter copies a list of fields to a new record and process this new record, discarding the original one.

"Matches" configuration is a list of list of strings in the form ["fieldName", "value1", "value2", ...].
The first element identifies the record field to check against the other elements of the list, using the OR condition (if any of the values matches the field, the condition is passed).
Multiple entries to "Matches" are evaluated as AND, which means that all conditions must pass to match the record.

Example:

Matches = [["timestamp", "value1", "value2"], ["source", "host1"], ["target", "host2", "host3"]]

Is equivalent to:

((timestamp == "value1" OR timestamp == "value2") AND (source == "host1") AND (target == "host2" OR target == "host3"))

If the provided conditions don't match the record, then the original record is discarded when DiscardNotMatching is true or is sent down the filter chain unchanged otherwise.

The "Fields" configuration lists the fields to copy from the original record to the new one, in case the record matches the conditions.
The new record will only contain the values of these fields, all other fields will be empty.
`

var PartialCloneDesc = baker.FilterDesc{
	Name:   "PartialClone",
	New:    NewPartialClone,
	Config: &PartialCloneConfig{},
	Help:   subRecordHelp,
}

type PartialCloneConfig struct {
	Matches            [][]string `help:"Conditions used to identify records to apply the filter to. See help for details. Missing configuration matches all records" required:"false"`
	DiscardNotMatching bool       `help:"Discard or maintain the records not matching the conditions" required:"false" default:"false"`
	Fields             []string   `help:"Fields that must be copied to the new line" required:"true"`
}

type PartialClone struct {
	numProcessedLines int64
	numFilteredLines  int64

	matches            map[baker.FieldIndex][][]byte
	discardNotMatching bool
	fieldIdx           []baker.FieldIndex
}

func NewPartialClone(cfg baker.FilterParams) (baker.Filter, error) {
	if cfg.DecodedConfig == nil {
		cfg.DecodedConfig = &PartialCloneConfig{}
	}
	dcfg := cfg.DecodedConfig.(*PartialCloneConfig)

	fieldIdx := []baker.FieldIndex{}
	for _, f := range dcfg.Fields {
		idx, ok := cfg.FieldByName(f)
		if !ok {
			return nil, fmt.Errorf("Can't resolve field name %s", f)
		}
		fieldIdx = append(fieldIdx, idx)
	}
	matches := make(map[baker.FieldIndex][][]byte)

	for _, f := range dcfg.Matches {
		idx, ok := cfg.FieldByName(f[0])
		if !ok {
			return nil, fmt.Errorf("Can't resolve field name %ss", f[0])
		}
		for i := 1; i < len(f); i++ {
			matches[idx] = append(matches[idx], []byte(f[i]))
		}

	}
	ut := &PartialClone{
		fieldIdx:           fieldIdx,
		discardNotMatching: dcfg.DiscardNotMatching,
		matches:            matches,
	}
	return ut, nil
}

func (s *PartialClone) Stats() baker.FilterStats {
	return baker.FilterStats{
		NumProcessedLines: atomic.LoadInt64(&s.numProcessedLines),
		NumFilteredLines:  atomic.LoadInt64(&s.numFilteredLines),
	}
}

func (s *PartialClone) Process(r baker.Record, next func(baker.Record)) {
	atomic.AddInt64(&s.numProcessedLines, 1)

	if !s.recordMatch(r) {
		if s.discardNotMatching {
			atomic.AddInt64(&s.numFilteredLines, 1)
			return
		}
		next(r)
		return
	}

	var l2 baker.LogLine
	for _, idx := range s.fieldIdx {
		l2.Set(idx, r.Get(idx))
	}
	next(&l2)
}

func (s *PartialClone) recordMatch(r baker.Record) bool {
	for k, values := range s.matches {
		fvalue := r.Get(k)
		m := false
		for _, v := range values {
			if bytes.Equal(v, fvalue) {
				m = true
				break
			}
		}
		if m == false {
			return false
		}
	}
	return true
}
