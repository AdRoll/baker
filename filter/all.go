package filter

import (
	"github.com/AdRoll/baker"
)

// AllFilters returns the list of all supported filters.
func AllFilters() []baker.FilterDesc {
	return []baker.FilterDesc{
		ClauseFilterDesc,
	}
}
