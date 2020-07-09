package filter

import (
	"testing"

	"github.com/AdRoll/baker"
)

func assertPanic(t *testing.T, f func()) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()
	f()
}

var field2idx = map[string]baker.FieldIndex{
	"f0": 0,
	"f1": 1,
	"f2": 2,
}

func fieldByName(name string) (baker.FieldIndex, bool) {
	idx, ok := field2idx[name]
	return idx, ok
}

func TestClauseParser(t *testing.T) {
	cf := &ClauseFilter{
		fieldByName: fieldByName,
	}
	// This just tests that the parser doesn't crash with various correctly formatted clauses
	cf.parseClause("(and (not (f0 value0)) (f0 notvalue0) (f1 notvalue1))")
	cf.parseClause("")
	cf.parseClause("(or (f0 notvalue0))")
	cf.parseClause("(and)")
	cf.parseClause("(and (or (f0 value0)   (not (f0  notvalue0 ))  ))")
}

func TestClausesMatchCorrectly(t *testing.T) {
	line1 := []byte("value0\x1evalue1\x1evalue3")
	logline := baker.NewLogLineFromText(line1)

	cfg := baker.FilterParams{
		ComponentParams: baker.ComponentParams{
			FieldByName:   fieldByName,
			DecodedConfig: &ClauseFilterConfig{Clause: "(and (f0 value0) (f1 value1))"},
		},
	}
	filter, _ := NewClauseFilter(cfg)
	matched_clause := false

	filter.Process(logline, func(baker.Record) { matched_clause = true })
	if matched_clause == false {
		t.Errorf("Clause filter filtered a line it should not have.")
	}

	// Sabotage f1, should no longer match
	cfg.DecodedConfig = &ClauseFilterConfig{Clause: "(and (f0 value0) (f1 notvalue1))"}
	filter, _ = NewClauseFilter(cfg)
	matched_clause = false

	filter.Process(logline, func(baker.Record) { matched_clause = true })
	if matched_clause == true {
		t.Errorf("Clause filter filtered a line it should not have.")
	}

	// Switch and to or, now value0 f0 should match
	cfg.DecodedConfig = &ClauseFilterConfig{Clause: "(or (f0 value0) (f1 notvalue1))"}
	filter, _ = NewClauseFilter(cfg)
	matched_clause = false

	filter.Process(logline, func(baker.Record) { matched_clause = true })
	if matched_clause == false {
		t.Errorf("Clause filter filtered a line it should not have.")
	}

	// Should match non-pxls with the adgroup, so should match
	cfg.DecodedConfig = &ClauseFilterConfig{Clause: "(and (not (f0 notvalue0)) (f1 value1))"}
	filter, _ = NewClauseFilter(cfg)
	matched_clause = false

	filter.Process(logline, func(baker.Record) { matched_clause = true })
	if matched_clause == false {
		t.Errorf("Clause filter filtered a line it should not have.")
	}
	// Should match non-value0s with the adgroup, so should NOT match
	cfg.DecodedConfig = &ClauseFilterConfig{Clause: "(and (not (f0 value0)) (f1 value1))"}
	filter, _ = NewClauseFilter(cfg)
	matched_clause = false

	filter.Process(logline, func(baker.Record) { matched_clause = true })
	if matched_clause == true {
		t.Errorf("Clause filter filtered a line it should not have.")
	}

	// Anything should pass empty clause
	cfg.DecodedConfig = &ClauseFilterConfig{Clause: ""}
	filter, _ = NewClauseFilter(cfg)
	matched_clause = false

	filter.Process(logline, func(baker.Record) { matched_clause = true })
	if matched_clause == false {
		t.Errorf("Clause filter filtered a line it should not have.")
	}
}
