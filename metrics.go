package baker

import (
	"fmt"
	"time"
)

// A MetricsBag is collection of metrics, those metrics are reported by
// every Baker components, through their Stats method.
// Stats() is called once per second, and contains a MetricsBag filled
// with values relative to that last second.
type MetricsBag map[string]interface{}

// AddRawCounter adds a counter. A counter is a cumulative metrics that can only increase.
// value must be positive.
func (bag MetricsBag) AddRawCounter(name string, value int64) {
	bag["c:"+name] = value
}

// AddDeltaCounter adds a new delta increment for a counter. The increment represents
// the variation of the counter in the current time interval.
// delta must be positive.
func (bag MetricsBag) AddDeltaCounter(name string, delta int64) {
	bag["d:"+name] = delta
}

// AddGauge adds a new snapshot of a value. A Gauge represents a single numerical
// data point that can arbitrarily go up and down.
func (bag MetricsBag) AddGauge(name string, value float64) {
	bag["g:"+name] = value
}

// AddHistogram adds a set of values to track their statistical distribution.
// A histogram samples observations and counts them in different 'buckets' in order
// to track and show the statistical distribution of a set of values.
func (bag MetricsBag) AddHistogram(name string, values []float64) {
	bag["h:"+name] = values
}

// AddTimings adds a set of timings to track their statistical distribution.
// A histogram samples observations and counts them in different 'buckets'. Timings
// is basically an histogram but allows to sample values of type time.Duration.
func (bag MetricsBag) AddTimings(name string, values []time.Duration) {
	bag["t:"+name] = values
}

// Merge merges another MetricsBag into this 'bag'.
func (bag MetricsBag) Merge(other MetricsBag) {
	for key, val := range other {
		switch key[0] {
		case 'c', 'd':
			// Counters and deltas should be summed
			if _, ok := bag[key]; !ok {
				bag[key] = int64(0)
			}
			bag[key] = bag[key].(int64) + val.(int64)
		case 'g':
			// Gauges must be averaged
			if _, ok := bag[key]; !ok {
				bag[key] = val.(float64)
			}
			bag[key] = (bag[key].(float64) + val.(float64)) / 2

		case 'h':
			// Histograms are concatenated
			if _, ok := bag[key]; !ok {
				bag[key] = []float64{}
			}
			bag[key] = append(bag[key].([]float64), val.([]float64)...)

		case 't':
			// timings are concatenated
			if _, ok := bag[key]; !ok {
				bag[key] = []time.Duration{}
			}
			bag[key] = append(bag[key].([]time.Duration), val.([]time.Duration)...)

		default:
			panic(fmt.Errorf("unsupported key %q", key))
		}
	}
}
