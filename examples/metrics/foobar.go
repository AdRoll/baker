package main

import (
	"time"

	"github.com/AdRoll/baker"
	"github.com/sirupsen/logrus"
)

// fooBarDesc describes how to plug fooBar as a metrics backend to Baker.
var fooBarDesc = baker.MetricsDesc{
	Name:   "FooBar",
	Config: &fooBarMetricsCfg{},
	New:    newFooBarMetrics,
}

type fooBarMetricsCfg struct {
	Host string
	Port int
}

var _ baker.MetricsClient = fooBarMetrics{}

type fooBarMetrics struct{}

func newFooBarMetrics(icfg interface{}) (baker.MetricsClient, error) {
	cfg := icfg.(*fooBarMetricsCfg)
	logrus.WithFields(logrus.Fields{"host": cfg.Host, "port": cfg.Port}).Info("FooBar metrics client instantiated")

	return fooBarMetrics{}, nil
}

// Gauge sets the value of a metric of type gauge. A Gauge represents a
// single numerical data point that can arbitrarily go up and down.
func (fooBarMetrics) Gauge(name string, value float64) {}

// GaugeWithTags sets the value of a metric of type gauge and associates
// that value with a set of tags.
func (fooBarMetrics) GaugeWithTags(name string, value float64, tags []string) {}

// RawCount sets the value of a metric of type counter. A counter is a
// cumulative metrics that can only increase. RawCount sets the current
// value of the counter.
func (fooBarMetrics) RawCount(name string, value int64) {}

// RawCountWithTags sets the value of a metric or type counter and associates
// that value with a set of tags.
func (fooBarMetrics) RawCountWithTags(name string, value int64, tags []string) {}

// DeltaCount increments the value of a metric of type counter by delta.
// delta must be positive.
func (fooBarMetrics) DeltaCount(name string, delta int64) {}

// DeltaCountWithTags increments the value of a metric or type counter and
// associates that value with a set of tags.
func (fooBarMetrics) DeltaCountWithTags(name string, delta int64, tags []string) {}

// Histogram adds a sample to a metric of type histogram. A histogram
// samples observations and counts them in different 'buckets' in order
// to track and show the statistical distribution of a set of values.
func (fooBarMetrics) Histogram(name string, value float64) {}

// HistogramWithTags adds a sample to an histogram and associates that
// sample with a set of tags.
func (fooBarMetrics) HistogramWithTags(name string, value float64, tags []string) {}

// Duration adds a duration to a metric of type histogram. A histogram
// samples observations and counts them in different 'buckets'. Duration
// is basically an histogram but allows to sample values of type time.Duration.
func (fooBarMetrics) Duration(name string, value time.Duration) {}

// DurationWithTags adds a duration to an histogram and associates that
// duration with a set of tags.
func (fooBarMetrics) DurationWithTags(name string, value time.Duration, tags []string) {}

// Close releases resources allocated by the metrics client such as
// connections or files and flushes potentially buffered data that has not
// been processed yet. Once closed the MetricsClient shoudld not be reused.
func (fooBarMetrics) Close() error { return nil }
