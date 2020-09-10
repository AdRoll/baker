// Package datadog provides types and functions to export  metrics
// and logs to Datadog via a statds client.
package datadog

import (
	"fmt"
	"sync"
	"time"

	"github.com/DataDog/datadog-go/statsd"
	log "github.com/sirupsen/logrus"

	"github.com/AdRoll/baker"
)

// Desc describes the Datadog metrics client inteface.
var Desc = baker.MetricsDesc{
	Name:   "Datadog",
	Config: &Config{},
	New:    newClient,
}

// Config is the configuration of the Datadog metrics client.
type Config struct {
	Prefix   string   // Prefix is the prefix of all metric names. defaults to baker.
	Host     string   // Host is the address (host:port) of the statsd host to send log to (in UDP). defaults to 127.0.0.1:8125.
	Tags     []string // Tags is the list of tags to attach to all metrics.
	SendLogs bool     // SendLogs indicates whether logs should be sent (as statds events) to the statsd client at Host.
}

// Client allows to instrument code and export the metrics to a dogstatds client.
type Client struct {
	dog *statsd.Client

	mu       sync.Mutex
	counters map[string]int64
}

// newClient creates a baker.MetricsClient that pushes to the datadog server using
// the dogstatsd format. All exported metrics will have a name prepended with
// the given prefix and will be tagged with the provided set of tags.
func newClient(icfg interface{}) (baker.MetricsClient, error) {
	cfg := icfg.(*Config)

	if cfg.Prefix == "" {
		cfg.Prefix = "baker."
	}

	if cfg.Host == "" {
		cfg.Host = "127.0.0.1:8125"
	}

	dog, err := statsd.NewBuffered(cfg.Host, 256)
	if err != nil {
		return nil, fmt.Errorf("can't create datadog metrics client: %s", err)
	}
	dog.Namespace = cfg.Prefix
	dog.Tags = cfg.Tags

	dd := &Client{
		dog:      dog,
		counters: make(map[string]int64),
	}

	if cfg.SendLogs {
		// Add Logrus hook converting log messages in statds events
		log.AddHook(NewHook(log.GetLevel(), dog, cfg.Host))
	}

	return dd, nil
}

// Gauge sets the value of a metric of type gauge. A Gauge represents a
// single numerical data point that can arbitrarily go up and down.
func (c *Client) Gauge(name string, value float64) {
	if c.dog != nil {
		c.dog.Gauge(name, value, nil, 1)
	}
}

// DeltaCount increments the value of a metric of type counter by delta.
// delta must be positive.
func (c *Client) DeltaCount(name string, delta int64) {
	if c.dog != nil {
		c.dog.Count(name, delta, nil, 1)
	}
}

// RawCount sets the value of a metric of type counter. A counter is a
// cumulative metrics that can only increase. RawCount sets the current
// value of the counter.
func (c *Client) RawCount(name string, value int64) {
	if c.dog != nil {
		c.mu.Lock()
		delta := value - c.counters[name]

		if delta < 0 {
			delta = 0
		}
		c.counters[name] = value
		c.mu.Unlock()

		c.dog.Count(name, delta, nil, 1)
	}
}

// Histogram adds a sample to a metric of type histogram. A histogram
// samples observations and counts them in different 'buckets' in order
// to track and show the statistical distribution of a set of values.
//
// In Datadog, this is shown as an 'Histogram', a DogStatsd metric type on
// which percentiles, mean and other info are calculated.
// see https://docs.datadoghq.com/developers/dogstatsd/data_types/#histograms
func (c *Client) Histogram(name string, value float64) {
	if c.dog != nil {
		c.dog.Histogram(name, value, nil, 1)
	}
}

// Duration adds a duration to a metric of type histogram. A histogram
// samples observations and counts them in different 'buckets'. Duration
// is basically an histogram but allows to sample values of type time.Duration.
//
// In Datadog, this is shown as a 'Timer', an implementation of an 'Histogram'
// DogStatsd  metric type, on which percentiles, mean and other info are calculated.
// see https://docs.datadoghq.com/developers/dogstatsd/data_types/#timers
func (c *Client) Duration(name string, value time.Duration) {
	if c.dog != nil {
		c.dog.TimeInMilliseconds(name, float64(value/time.Millisecond), nil, 1)
	}
}

// GaugeWithTags sets the value of a metric of type gauge and associates
// that value with a set of tags.
func (c *Client) GaugeWithTags(name string, value float64, tags []string) {
	if c.dog != nil {
		c.dog.Gauge(name, value, tags, 1)
	}
}

// DeltaCountWithTags increments the value of a metric or type counter and
// associates that value with a set of tags.
func (c *Client) DeltaCountWithTags(name string, delta int64, tags []string) {
	if c.dog != nil {
		c.dog.Count(name, delta, tags, 1)
	}
}

// RawCountWithTags sets the value of a metric or type counter and associates
// that value with a set of tags.
func (c *Client) RawCountWithTags(name string, value int64, tags []string) {
	if c.dog != nil {
		c.mu.Lock()
		delta := value - c.counters[name]

		if delta < 0 {
			delta = 0
		}
		c.counters[name] = value
		c.mu.Unlock()
		c.dog.Count(name, delta, tags, 1)
	}
}

// HistogramWithTags adds a sample to an histogram and associates that
// sample with a set of tags.
func (c *Client) HistogramWithTags(name string, value float64, tags []string) {
	if c.dog != nil {
		c.dog.Histogram(name, value, tags, 1)
	}
}

// DurationWithTags adds a duration to an histogram and associates that
// duration with a set of tags.
func (c *Client) DurationWithTags(name string, value time.Duration, tags []string) {
	if c.dog != nil {
		c.dog.TimeInMilliseconds(name, float64(value/time.Millisecond), tags, 1)
	}
}
