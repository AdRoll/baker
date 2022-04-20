package testutil

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/AdRoll/baker"
)

// MockMetricsDesc describes the MockMetrics metrics client.
var MockMetricsDesc = baker.MetricsDesc{
	Name:   "MockMetrics",
	Config: &struct{}{},
	New:    newMockMetrics,
}

// MockMetrics is a metrics client to be used in tests only, which stores single
// calls made to the set of methods implementing the baker.MetricsClient
// interface, and sort them so that they're easy to compare mechanically, in
// tests.
type MockMetrics struct {
	buf bytes.Buffer
}

func newMockMetrics(_ interface{}) (baker.MetricsClient, error) { return &MockMetrics{}, nil }

// PublishedMetrics returns a list of strings, each of which represent arguments
// and method of calls to methods of the baker.MetricsClient interface. Prefix
// can be used to select a subset of calls, or all of them (with ""). Go runtime
// metrics are ignored.
func (m *MockMetrics) PublishedMetrics(prefix string) []string {
	keep := make([]string, 0)
	for _, s := range strings.Split(m.buf.String(), "\n") {
		if len(strings.TrimSpace(s)) != 0 && !strings.Contains(s, "name=runtime.") {
			if len(prefix) == 0 || strings.HasPrefix(s, prefix) {
				keep = append(keep, s)
			}
		}
	}

	sort.Strings(keep)
	return keep
}

func (m *MockMetrics) Gauge(name string, value float64) {
	fmt.Fprintf(&m.buf, "gauge|name=%s|value=%v\n", name, value)
}
func (m *MockMetrics) RawCount(name string, value int64) {
	fmt.Fprintf(&m.buf, "rawcount|name=%s|value=%v\n", name, value)
}
func (m *MockMetrics) DeltaCount(name string, delta int64) {
	fmt.Fprintf(&m.buf, "delta|name=%s|value=%v\n", name, delta)
}
func (m *MockMetrics) Histogram(name string, value float64) {
	fmt.Fprintf(&m.buf, "hist|name=%s|value=%v\n", name, value)
}
func (m *MockMetrics) Duration(name string, value time.Duration) {
	fmt.Fprintf(&m.buf, "duration|name=%s|value=%v\n", name, value)
}

func (m *MockMetrics) GaugeWithTags(name string, value float64, tags []string) {
	for _, t := range tags {
		fmt.Fprintf(&m.buf, "gauge|name=%s|value=%v|tag=%s\n", name, value, t)
	}
}
func (m *MockMetrics) RawCountWithTags(name string, value int64, tags []string) {
	for _, t := range tags {
		fmt.Fprintf(&m.buf, "rawcount|name=%s|value=%v|tag=%s\n", name, value, t)
	}
}
func (m *MockMetrics) DeltaCountWithTags(name string, delta int64, tags []string) {
	for _, t := range tags {
		fmt.Fprintf(&m.buf, "delta|name=%s|value=%v|tag=%s\n", name, delta, t)
	}
}
func (m *MockMetrics) HistogramWithTags(name string, value float64, tags []string) {
	for _, t := range tags {
		fmt.Fprintf(&m.buf, "hist|name=%s|value=%v|tag=%s\n", name, value, t)
	}
}
func (m *MockMetrics) DurationWithTags(name string, value time.Duration, tags []string) {
	for _, t := range tags {
		fmt.Fprintf(&m.buf, "duration|name=%s|value=%v|tag=%s\n", name, value, t)
	}
}
func (m *MockMetrics) Close() error { return nil }
