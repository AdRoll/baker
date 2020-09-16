package metrics

import (
	"github.com/AdRoll/baker"
	"github.com/AdRoll/baker/metrics/datadog"
)

// All is the list of all metrics client supported by Baker.
var All = []baker.MetricsDesc{
	datadog.Desc,
}
