package metrics

import (
	"fmt"
	"sync"
	"time"

	"github.com/DataDog/datadog-go/statsd"
	log "github.com/sirupsen/logrus"
)

type Datadog struct {
	dog      *statsd.Client
	basetags []string

	mu       sync.Mutex
	counters map[string]int64
}

// NewDatadogClient creates a Client that pushes to the datadog server using
// the dogstatsd format. All exported metrics will have a name prepended with
// the given prefix and will be tagged with the provided set of tags.
func NewDatadogClient(host, prefix string, tags []string) (*Datadog, error) {
	dog, err := statsd.NewBuffered(host, 256)
	if err != nil {
		return nil, fmt.Errorf("can't create datadog metrics client: %s", err)
	}
	dog.Namespace = prefix

	dd := &Datadog{
		dog:      dog,
		basetags: tags,
		counters: make(map[string]int64),
	}
	return dd, nil
}

// Client returns the global statsd client.
// TODO(arl): check if and where it's used, but in theory we should remove this
func Client() *statsd.Client { return dog }

// Gauge sets the value of a metric of type gauge. A Gauge represents a
// single numerical data point that can arbitrarily go up and down.
func (dd *Datadog) Gauge(name string, value float64) {
	if dd.dog != nil {
		dd.dog.Gauge(name, value, dd.basetags, 1)
	}
}

// DeltaCount increments the value of a metric of type counter by delta.
// delta must be positive.
func (dd *Datadog) DeltaCount(name string, delta int64) {
	if dd.dog != nil {
		dd.dog.Count(name, delta, dd.basetags, 1)
	}
}

// RawCount sets the value of a metric of type counter. A counter is a
// cumulative metrics that can only increase. RawCount sets the current
// value of the counter.
func (dd *Datadog) RawCount(name string, value int64) {
	if dd.dog != nil {
		dd.mu.Lock()
		delta := value - dd.counters[name]

		// TODO: Once the sources of the weird data have been tracked
		// down, instead of crashing we should just submit a delta of
		// 0.
		if delta < 0 {
			log.Fatalf("encountered a negative delta! a metric of name '%s' that should always be increasing have decreased from last collection. old value %d, new value %d, delta %d\n", name, dd.counters[name], value, delta)
		}
		dd.counters[name] = value
		dd.mu.Unlock()

		dd.dog.Count(name, delta, dd.basetags, 1)
	}
}

// Histogram adds a sample to a metric of type histogram. A histogram
// samples observations and counts them in different 'buckets' in order
// to track and show the statistical distribution of a set of values.
//
// In Datadog, this is shown as an 'Histogram', a DogStatsd metric type on
// which percentiles, mean and other info are calculated.
// see https://docs.datadoghq.com/developers/dogstatsd/data_types/#histograms
func (dd *Datadog) Histogram(name string, value float64) {
	if dd.dog != nil {
		dd.dog.Histogram(name, value, dd.basetags, 1)
	}
}

// Duration adds a duration to a metric of type histogram. A histogram
// samples observations and counts them in different 'buckets'. Duration
// is basically an histogram but allows to sample values of type time.Duration.
//
// In Datadog, this is shown as a 'Timer', an implementation of an 'Histogram'
// DogStatsd  metric type, on which percentiles, mean and other info are calculated.
// see https://docs.datadoghq.com/developers/dogstatsd/data_types/#timers
func (dd *Datadog) Duration(name string, value time.Duration) {
	if dd.dog != nil {
		dd.dog.TimeInMilliseconds(name, float64(value/time.Millisecond), dd.basetags, 1)
	}
}

// GaugeWithTags sets the value of a metric of type gauge and associates
// that value with a set of tags.
func (dd *Datadog) GaugeWithTags(name string, value float64, tags []string) {
	if dd.dog != nil {
		dd.dog.Gauge(name, value, append(dd.basetags, tags...), 1)
	}
}

// DeltaCountWithTags increments the value of a metric or type counter and
// associates that value with a set of tags.
func (dd *Datadog) DeltaCountWithTags(name string, delta int64, tags []string) {
	if dd.dog != nil {
		dd.dog.Count(name, delta, append(tags, dd.basetags...), 1)
	}
}

// RawCountWithTags sets the value of a metric or type counter and associates
// that value with a set of tags.
func (dd *Datadog) RawCountWithTags(name string, value int64, tags []string) {
	if dd.dog != nil {
		dd.mu.Lock()
		delta := value - dd.counters[name]

		// TODO: Once the sources of the weird data have been tracked
		// down, instead of crashing we should just submit a delta of
		// 0.
		if delta < 0 {
			log.Fatalf("encountered a negative delta! a metric of name '%s' that should always be increasing have decreased from last collection. old value %d, new value %d, delta %d\n", name, dd.counters[name], value, delta)
		}
		dd.counters[name] = value
		dd.mu.Unlock()
		dd.dog.Count(name, delta, append(dd.basetags, tags...), 1)
	}
}

// HistogramWithTags adds a sample to an histogram and associates that
// sample with a set of tags.
func (dd *Datadog) HistogramWithTags(name string, value float64, tags []string) {
	if dd.dog != nil {
		dd.dog.Histogram(name, value, append(dd.basetags, tags...), 1)
	}
}

// DurationWithTags adds a duration to an histogram and associates that
// duration with a set of tags.
func (dd *Datadog) DurationWithTags(name string, value time.Duration, tags []string) {
	if dd.dog != nil {
		dd.dog.TimeInMilliseconds(name, float64(value/time.Millisecond), append(dd.basetags, tags...), 1)
	}
}

// // DurationMsWithTags tracks time durations.
// //
// // Same as DurationWithTags but avoid a conversion to float64 when the user value
// // is already a float in milliseconds.
// func (dd *Datadog) DurationMsWithTags(name string, ms float64, xtags []string) {
// 	if dog != nil {
// 		dog.TimeInMilliseconds(name, ms, append(tags, xtags...), 1)
// 	}
// }
