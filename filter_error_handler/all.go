// Package filter_error_handler provides error handler components for filters.
package filter_error_handler

import (
	"github.com/AdRoll/baker"
)

// All is the list of all baker filter error handlers.
var All = []baker.FilterErrorHandlerDesc{
	ClearFieldsDesc,
	LogDesc,
}
