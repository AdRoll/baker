package filter

import (
	"fmt"
	"sync/atomic"

	"github.com/AdRoll/baker"
)

var PartialCloneDesc = baker.FilterDesc{
	Name:   "PartialClone",
	New:    NewPartialClone,
	Config: &PartialCloneConfig{},
	Help:   "Copy a list of fields to a new record and process this new record, discarding the original one",
}

type PartialCloneConfig struct {
	Fields []string `help:"Fields that must be copied to the new line" required:"true"`
}

type PartialClone struct {
	numProcessedLines int64
	numFilteredLines  int64

	fieldIdx     []baker.FieldIndex
	createRecord func() baker.Record
}

func NewPartialClone(cfg baker.FilterParams) (baker.Filter, error) {
	dcfg := cfg.DecodedConfig.(*PartialCloneConfig)

	if len(dcfg.Fields) == 0 {
		return nil, fmt.Errorf("PartialClone: add at least one field")
	}

	fieldIdx := []baker.FieldIndex{}
	for _, f := range dcfg.Fields {
		idx, ok := cfg.FieldByName(f)
		if !ok {
			return nil, fmt.Errorf("can't resolve field name %s", f)
		}
		fieldIdx = append(fieldIdx, idx)
	}
	ut := &PartialClone{
		fieldIdx:     fieldIdx,
		createRecord: cfg.CreateRecord,
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

	l2 := s.createRecord()
	for _, idx := range s.fieldIdx {
		l2.Set(idx, r.Get(idx))
	}
	next(l2)
}
