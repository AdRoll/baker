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
	CryptDesc,
	DedupDesc,
	ExpandJSONDesc,
	ExpandListDesc,
	FormatTimeDesc,
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
	TruncateDesc,
}
