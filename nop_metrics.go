package baker

import "time"

var _ MetricsClient = NopMetrics{}

// NopMetrics implements a metrics.Client that does nothing.
type NopMetrics struct{}

func (NopMetrics) Gauge(name string, value float64)                                 {}
func (NopMetrics) GaugeWithTags(name string, value float64, tags []string)          {}
func (NopMetrics) RawCount(name string, value int64)                                {}
func (NopMetrics) RawCountWithTags(name string, value int64, tags []string)         {}
func (NopMetrics) DeltaCount(name string, delta int64)                              {}
func (NopMetrics) DeltaCountWithTags(name string, delta int64, tags []string)       {}
func (NopMetrics) Histogram(name string, value float64)                             {}
func (NopMetrics) HistogramWithTags(name string, value float64, tags []string)      {}
func (NopMetrics) Duration(name string, value time.Duration)                        {}
func (NopMetrics) DurationWithTags(name string, value time.Duration, tags []string) {}
func (NopMetrics) Close() error                                                     { return nil }
