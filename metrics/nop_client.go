package metrics

import "time"

// NopClient implements a Client that does nothing.
type NopClient struct{}

func (NopClient) Gauge(name string, value float64)                                 {}
func (NopClient) GaugeWithTags(name string, value float64, tags []string)          {}
func (NopClient) RawCount(name string, value int64)                                {}
func (NopClient) RawCountWithTags(name string, value int64, tags []string)         {}
func (NopClient) DeltaCount(name string, delta int64)                              {}
func (NopClient) DeltaCountWithTags(name string, delta int64, tags []string)       {}
func (NopClient) Histogram(name string, value float64)                             {}
func (NopClient) HistogramWithTags(name string, value float64, tags []string)      {}
func (NopClient) Duration(name string, value time.Duration)                        {}
func (NopClient) DurationWithTags(name string, value time.Duration, tags []string) {}
