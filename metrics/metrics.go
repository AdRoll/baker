package metrics

import (
	"sync"
	"time"

	"github.com/DataDog/datadog-go/statsd"
	log "github.com/sirupsen/logrus"
)

var (
	dog          *statsd.Client
	counters     map[string]int64
	countersLock sync.Mutex
	tags         []string
)

func Init(host, prefix string, inputtags []string) {
	var err error

	dog, err = statsd.NewBuffered(host, 256)
	if err != nil {
		log.Fatal(err)
	}
	tags = inputtags

	dog.Namespace = prefix
	counters = make(map[string]int64)
}

// Client returns the global statsd client.
func Client() *statsd.Client { return dog }

func Gauge(name string, value float64) {
	if dog != nil {
		dog.Gauge(name, value, tags, 1)
	}
}

func DeltaCount(name string, delta int64) {
	if dog != nil {
		dog.Count(name, delta, tags, 1)
	}
}

func RawCount(name string, value int64) {
	if dog != nil {
		countersLock.Lock()
		delta := value - counters[name]

		// TODO: Once the sources of the weird data have been tracked
		// down, instead of crashing we should just submit a delta of
		// 0.
		if delta < 0 {
			log.Fatalf("encountered a negative delta! a metric of name '%s' that should always be increasing have decreased from last collection. old value %d, new value %d, delta %d\n", name, counters[name], value, delta)
		}
		counters[name] = value
		countersLock.Unlock()
		dog.Count(name, delta, tags, 1)
	}
}

// Histogram tracks the statistical distribution of a set of values.
//
// It is shown as an 'Histogram', a DogStatsd metric type on which
// percentiles, mean and other info are calculated.
// see https://docs.datadoghq.com/developers/dogstatsd/data_types/#histograms
func Histogram(name string, value float64) {
	if dog != nil {
		dog.Histogram(name, value, tags, 1)
	}
}

// Duration tracks time durations.
//
// It is shown as a 'Timer', an implementation of an 'Histogram' DogStatsd
// metric type, on which percentiles, mean and other info are calculated.
// see https://docs.datadoghq.com/developers/dogstatsd/data_types/#timers
func Duration(name string, value time.Duration) {
	if dog != nil {
		dog.TimeInMilliseconds(name, float64(value/time.Millisecond), tags, 1)
	}
}

func GaugeWithTags(name string, value float64, xtags []string) {
	if dog != nil {
		dog.Gauge(name, value, append(tags, xtags...), 1)
	}
}

func DeltaCountWithTags(name string, delta int64, xtags []string) {
	if dog != nil {
		dog.Count(name, delta, append(tags, xtags...), 1)
	}
}

func RawCountWithTags(name string, value int64, xtags []string) {
	if dog != nil {
		countersLock.Lock()
		delta := value - counters[name]

		// TODO: Once the sources of the weird data have been tracked
		// down, instead of crashing we should just submit a delta of
		// 0.
		if delta < 0 {
			log.Fatalf("encountered a negative delta! a metric of name '%s' that should always be increasing have decreased from last collection. old value %d, new value %d, delta %d\n", name, counters[name], value, delta)
		}
		counters[name] = value
		countersLock.Unlock()
		dog.Count(name, delta, append(tags, xtags...), 1)
	}
}

// HistogramWithTags tracks the statistical distribution of a set of values.
//
// It is shown as an 'Histogram', a DogStatsd metric type on which
// percentiles, mean and other info are calculated.
// see https://docs.datadoghq.com/developers/dogstatsd/data_types/#histograms
func HistogramWithTags(name string, value float64, xtags []string) {
	if dog != nil {
		dog.Histogram(name, value, append(tags, xtags...), 1)
	}
}

// DurationWithTags tracks time durations.
//
// It is shown as a 'Timer', an implementation of an 'Histogram' DogStatsd
// metric type, on which percentiles, mean and other info are calculated.
// see https://docs.datadoghq.com/developers/dogstatsd/data_types/#timers
func DurationWithTags(name string, value time.Duration, xtags []string) {
	if dog != nil {
		dog.TimeInMilliseconds(name, float64(value/time.Millisecond), append(tags, xtags...), 1)
	}
}

// DurationMsWithTags tracks time durations.
//
// Same as DurationWithTags but avoid a conversion to float64 when the user value
// is already a float in milliseconds.
func DurationMsWithTags(name string, ms float64, xtags []string) {
	if dog != nil {
		dog.TimeInMilliseconds(name, ms, append(tags, xtags...), 1)
	}
}
