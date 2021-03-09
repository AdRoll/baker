package filter

import (
	"bytes"
	"strings"
	"sync/atomic"

	"github.com/nsf/sexp"
	log "github.com/sirupsen/logrus"

	"github.com/AdRoll/baker"
)

const clauseFilterHelpMsg = `
Discard records which do not match a clause given as a boolean S-expression. Check the filter documentation for some examples.

### ClauseFilter boolean expression format

This document describes the s-expression format used in ClauseFilter.

The format uses s-expressions. Empty string matches anything (i.e. all records
will pass the expression).

There are only three keywords: and, or, not

If an s-expression starts with any other name, it is assumed to be the name of
a field and it should be paired with the desired value to match against.

    Must match both X and Y to pass:
    (and X Y)

    You can use more than 2 arguments:
    (and X Y Z A B C)

    Must match either X or Y to pass:
    (or X Y)

    Must NOT match X to pass:
    (not X)

    Field must equal value to pass:
    (FIELD VALUE)

    example:
    (fieldName somevalue)

    Matches anything (because only one argument)
    (and X)

    Matches nothing
    (and)

    Matches anything
    (or)

Examples:

    (and (fieldName value1) (anotherFieldName value2))

    (or (fieldName value1) (fieldName value2))

	(not (or (fieldName value1) (fieldName value2)))

    (or
      (and (fieldName value1)
           (anotherFieldName value3))
      (and (fieldName value2)
           (anotherFieldName value4)))
`

// ClauseFilterDesc describes the ClauseFilter filter
var ClauseFilterDesc = baker.FilterDesc{
	Name:   "ClauseFilter",
	New:    NewClauseFilter,
	Config: &ClauseFilterConfig{},
	Help:   clauseFilterHelpMsg,
}

// ClauseFilterConfig describes the ClauseFilter filter config
type ClauseFilterConfig struct {
	Clause string `help:"Boolean formula describing which events to let through. If empty, let everything through."`
}

type ClauseFilter struct {
	cfg               *ClauseFilterConfig
	topClause         Clause
	numProcessedLines int64
	numFilteredLines  int64
	fieldByName       func(string) (baker.FieldIndex, bool)
	aaa               baker.FilterParams
}

type clauseType int

const (
	not_clause clauseType = iota
	and_clause
	or_clause
	atom_clause
	true_clause
	false_clause
)

func NewClauseFilter(cfg baker.FilterParams) (baker.Filter, error) {
	dcfg := cfg.DecodedConfig.(*ClauseFilterConfig)
	if dcfg.Clause == "" {
		log.Warn("ClauseFilter is being used but the Clause string is empty. This means everything will be passed through this filter.")
	}

	cf := &ClauseFilter{
		aaa:         cfg,
		cfg:         dcfg,
		fieldByName: cfg.FieldByName,
	}
	cf.topClause = cf.parseClause(dcfg.Clause)
	return cf, nil
}

// Will real sum types ever make it to go?
//
// Validness of the fields in this structure depend on clauseType
type Clause struct {
	clauseType         clauseType
	leftClause         *Clause
	rightClause        *Clause
	matchField         baker.FieldIndex
	matchFieldContents []byte
}

func (f *ClauseFilter) Process(l baker.Record, next func(baker.Record)) {
	atomic.AddInt64(&f.numProcessedLines, 1)
	if f.matchClause(l, &f.topClause) {
		next(l)
	} else {
		atomic.AddInt64(&f.numFilteredLines, 1)
	}
}

func (f *ClauseFilter) Stats() baker.FilterStats {
	return baker.FilterStats{
		NumProcessedLines: atomic.LoadInt64(&f.numProcessedLines),
		NumFilteredLines:  atomic.LoadInt64(&f.numFilteredLines),
	}
}

func (f *ClauseFilter) parseClauseSexp(node *sexp.Node) Clause {
	if node.Children != nil && (node.Children.Value == "and" || node.Children.Value == "or") {
		clause_type := or_clause
		if node.Children.Value == "and" {
			clause_type = and_clause
		}

		if node.NumChildren() == 1 {
			if node.Children.Value == "or" {
				return Clause{clauseType: true_clause}
			} else {
				return Clause{clauseType: false_clause}
			}
		}
		if node.NumChildren() == 2 {
			second, err := node.Nth(1)
			if err != nil {
				panic(err)
			}
			return f.parseClauseSexp(second)
		}
		if node.NumChildren() == 3 {
			first, err1 := node.Nth(1)
			if err1 != nil {
				panic(err1)
			}
			second, err2 := node.Nth(2)
			if err2 != nil {
				panic(err2)
			}
			lclause := f.parseClauseSexp(first)
			rclause := f.parseClauseSexp(second)
			return Clause{clauseType: clause_type,
				leftClause:  &lclause,
				rightClause: &rclause}
		}

		second, err2 := node.Nth(2)
		if err2 != nil {
			log.WithError(err2).Fatal("Cannot interpret child for and/or clause")
		}

		inner_node := sexp.Node{Value: node.Children.Value, Next: second}
		rnode := sexp.Node{Children: &inner_node}

		first, err1 := node.Nth(1)
		if err1 != nil {
			log.WithError(err1).Fatal("Cannot interpret child for and/or clause")
		}

		lclause := f.parseClauseSexp(first)
		rclause := f.parseClauseSexp(&rnode)

		return Clause{clauseType: clause_type,
			leftClause:  &lclause,
			rightClause: &rclause}
	}
	if node.NumChildren() == 2 && node.Children.Value == "not" {
		child_node, err := node.Nth(1)
		if err != nil {
			log.WithError(err).Fatal("Cannot interpret child for not-clause")
		}
		child := f.parseClauseSexp(child_node)
		return Clause{clauseType: not_clause,
			leftClause: &child}
	}
	if node.NumChildren() == 2 {
		field_node, err1 := node.Nth(0)
		value_node, err2 := node.Nth(1)
		if err1 != nil {
			log.WithError(err1).Fatal("Cannot interpret clause s-expression")
		}
		if err2 != nil {
			log.WithError(err2).Fatal("Cannot interpret clause s-expression")
		}

		field := field_node.Value
		value := value_node.Value

		idx, ok := f.fieldByName(field)
		if !ok {
			log.Fatal("No such field: ", field)
		}

		return Clause{clauseType: atom_clause,
			matchField:         idx,
			matchFieldContents: []byte(value)}
	}

	log.Fatal("Cannot interpret s-expression. Verify it's correctly written.")
	panic("Unreachable")
}

func (f *ClauseFilter) parseClause(ClauseRaw string) Clause {
	ClauseString := strings.TrimLeft(strings.TrimRight(ClauseRaw, " \r\n\t"), " \r\n\t")
	if len(ClauseString) == 0 {
		return Clause{clauseType: true_clause}
	}

	buf := bytes.NewBufferString(ClauseString)
	var ctx sexp.SourceContext
	file := ctx.AddFile("clause", 1)
	top, _ := sexp.Parse(buf, file)
	clause := top.Children

	return f.parseClauseSexp(clause)
}

func (f *ClauseFilter) matchClause(l baker.Record, clause *Clause) bool {
	switch clause.clauseType {
	case not_clause:
		result := f.matchClause(l, clause.leftClause)
		return !result
	case and_clause:
		result_left := f.matchClause(l, clause.leftClause)
		// Shortcircuit the 'and' if we can
		if !result_left {
			return false
		}
		result_right := f.matchClause(l, clause.rightClause)
		return result_right
	case or_clause:
		result_left := f.matchClause(l, clause.leftClause)
		// Shortcircuit the 'or' if we can
		if result_left {
			return true
		}
		result_right := f.matchClause(l, clause.rightClause)
		return result_right
	case atom_clause:
		log_field := l.Get(clause.matchField)
		return bytes.Equal(log_field, clause.matchFieldContents)
	case true_clause:
		return true
	case false_clause:
		return false
	}

	panic("case fell through somewhere that should not be possible")
}
