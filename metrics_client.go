package baker

import (
	"time"
)

// A MetricsClient allows to instrument components code and communicate the
// metrics to the metrics backend that is configured in Baker.
//
// New metrics backends must implement this interface and register their
// description as a MetricDesc in Components.Metrics. See ./examples/metrics.
type MetricsClient interface {

	// Gauge sets the value of a metric of type gauge. A Gauge represents a
	// single numerical data point that can arbitrarily go up and down.
	Gauge(name string, value float64)

	// GaugeWithTags sets the value of a metric of type gauge and associates
	// that value with a set of tags.
	GaugeWithTags(name string, value float64, tags []string)

	// RawCount sets the value of a metric of type counter. A counter is a
	// cumulative metrics that can only increase. RawCount sets the current
	// value of the counter.
	RawCount(name string, value int64)

	// RawCountWithTags sets the value of a metric or type counter and associates
	// that value with a set of tags.
	RawCountWithTags(name string, value int64, tags []string)

	// DeltaCount increments the value of a metric of type counter by delta.
	// delta must be positive.
	DeltaCount(name string, delta int64)

	// DeltaCountWithTags increments the value of a metric or type counter and
	// associates that value with a set of tags.
	DeltaCountWithTags(name string, delta int64, tags []string)

	// Histogram adds a sample to a metric of type histogram. A histogram
	// samples observations and counts them in different 'buckets' in order
	// to track and show the statistical distribution of a set of values.
	Histogram(name string, value float64)

	// HistogramWithTags adds a sample to an histogram and associates that
	// sample with a set of tags.
	HistogramWithTags(name string, value float64, tags []string)

	// Duration adds a duration to a metric of type histogram. A histogram
	// samples observations and counts them in different 'buckets'. Duration
	// is basically an histogram but allows to sample values of type time.Duration.
	Duration(name string, value time.Duration)

	// DurationWithTags adds a duration to an histogram and associates that
	// duration with a set of tags.
	DurationWithTags(name string, value time.Duration, tags []string)
}
