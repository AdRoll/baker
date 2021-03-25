// Package filter provides filter components.
package filter

import (
	"github.com/AdRoll/baker"
)

// All is the list of all baker filters.
var All []interface{}

func init() {
	// TODO(arl) sort by name
	for _, f := range AllFilters {
		All = append(All, f)
	}
	for _, f := range AllModifiers {
		All = append(All, f)
	}
}

var AllFilters = []baker.FilterDesc{
	ClauseFilterDesc,
	ClearFieldsDesc,
	ConcatenateDesc,
	DedupDesc,
	ExpandJSONDesc,
	ExpandListDesc,
	HashDesc,
	MetadataLastModifiedDesc,
	MetadataUrlDesc,
	NotNullDesc,
	PartialCloneDesc,
	RegexMatchDesc,
	ReplaceFieldsDesc,
	SetStringFromURLDesc,
	StringMatchDesc,
	TimestampDesc,
	TimestampRangeDesc,
}

// All is the list of all baker filters.
var AllModifiers = []baker.ModifierDesc{
	FormatTimeDesc,
}
