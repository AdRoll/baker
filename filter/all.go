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
	CountAndTagDesc,
	CryptDesc,
	DedupDesc,
	ExpandJSONDesc,
	ExpandListDesc,
	ExternalMatchDesc,
	FormatTimeDesc,
	HashDesc,
	MetadataLastModifiedDesc,
	MetadataUrlDesc,
	NotNullDesc,
	PartialCloneDesc,
	RegexMatchDesc,
	ReplaceFieldsDesc,
	SetStringFromURLDesc,
	SliceDesc,
	StringMatchDesc,
	TimestampDesc,
	TimestampRangeDesc,
	URLEscapeDesc,
	URLParamDesc,
}
