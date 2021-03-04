// Package filter provides filter components.
package filter

import (
	"github.com/AdRoll/baker"
)

// All is the list of all baker filters.
var All = []baker.FilterDesc{
	ClauseFilterDesc,
	ClearFieldsDesc,
	ConcatenateDesc,
	DedupDesc,
	ExpandJSONDesc,
	HashDesc,
	LUADesc,
	MetadataLastModifiedDesc,
	NotNullDesc,
	PartialCloneDesc,
	RegexMatchDesc,
	ReplaceFieldsDesc,
	SetStringFromURLDesc,
	StringMatchDesc,
	TimestampDesc,
	TimestampRangeDesc,
}
