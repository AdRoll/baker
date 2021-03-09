---
title: "Export metrics"
date: 2020-11-03
weight: 770
description: >
  How to export metrics from Baker
---

Baker can publish various kind of [metrics](/docs/core-concepts/#metrics) that may
be used to monitor a pipeline in execution. The metrics exported range from numbers
giving an high-level overview of the ongoing pipeline (total processed records, 
current speed in records per second, etc.) or per-component metrics such as the 
number of files read or written, to performance statistics published by the Go 
runtime in order to monitor lower level information (objects, memory, garbage 
collection, etc.).

## Configuring metrics in TOML

Baker configuration TOML files may have a `[metrics]` section dedicated to the 
configuration of a metrics client.
`[metrics.name]` specifies the metrics client to use, from the list of all registered 
[`baker.MetricsClient`](https://pkg.go.dev/github.com/AdRoll/baker#MetricsClient).
`[metrics.config]` specifies some configuration settings which are specific to 
the client you're using.

For example, this is what the `[metrics]` section would look like with the *Datadog*
metrics client:

```toml
[metrics]
name="datadog"

    [metrics.config]
    host="localhost:8125"                  # address of the dogstatsd client to which send metrics to
    prefix="myapp.baker."                  # prefix for all exported metric names
    send_logs=true                         # whether we should log messages (as Dogstatd events) or not 
    tags=["env:prod", "region:eu-west-1"]  # extra tags to associate to all exported metrics 
```

### Disabling metrics

If you don't want to publish any metrics, it's enough to not provide the `[metrics]` TOML 
section in Baker configuration file.


## Create a Custom MetricsClient

The [metrics example](https://github.com/AdRoll/baker/tree/main/examples/metrics) shows an
example implementation of
[`baker.MetricsClient`](https://pkg.go.dev/github.com/AdRoll/baker#MetricsClient)
and how to register it within Baker so that it can be selected in the
`[metrics.name]` TOML section.

In order to be selected from TOML, you must first register a 
[`baker.MetricsDesc`](https://pkg.go.dev/github.com/AdRoll/baker#MetricsDesc) 
instance within [`baker.Components`](https://pkg.go.dev/github.com/AdRoll/baker#Components).

```go
var fooBarDesc = baker.MetricsDesc{
	Name:   "MyMetrics",
	Config: &myyMetricsConfig{},
	New:    newMyMetrics,
}
```

where `newMyMetrics` is a constructor-like function receiving an `interface{}`,
which is guaranteed to be of the type of the `Config` field value. This function
should either return a ready to use
[`baker.MetricsClient`](https://pkg.go.dev/github.com/AdRoll/baker#MetricsClient)
or an error saying why it can't.

```go
func newMyMetrics(icfg interface{}) (baker.MetricsClient, error)
```

## How to expose statistics in a Component

All components need to implement a `Stats` method where they can expose metrics. Baker 
runtime calls the `Stats` method of each component once per second. The different component 
types support a set of predefined metrics. In particular, the following counters are defined:

- [Input](https://pkg.go.dev/github.com/AdRoll/baker#InputStats):
    - `NumProcessedLines`, the total number of processed records since the component creation
- [Filter](https://pkg.go.dev/github.com/AdRoll/baker#FilterStats):
    - `NumProcessedLines`, the total number of processed records since the component creation
    - `NumFilteredLines`, the number of discarded (i.e., filtered) records
- [Output](https://pkg.go.dev/github.com/AdRoll/baker#OutputStats):
    - `NumProcessedLines`, the total number of processed records since the component creation
    - `NumErrorLines`, the number of records that have produced an error
- [Upload](https://pkg.go.dev/github.com/AdRoll/baker#UploadStats):
    - `NumProcessedFiles`, the total number of processed files since the component creation
    - `NumErrorFiles`, the number of files that have produced an error

Due to historical reasons, these fields have the word _lines_ in them but they do 
mean the number of records.

The code example of 
[how-to-create-filter](https://getbaker.io/docs/how-tos/create_filter/#processing-records) 
shows how to correctly reports metrics in a custom Filter.

### Report custom metrics

Components may need to report custom metrics for monitoring some specific events. 
Baker supports two main ways to expose custom metrics, namely:

- return a 
[`baker.MetricsBag`](https://pkg.go.dev/github.com/AdRoll/baker#MetricsBag)
instance from the `Stats` method 
- directly use the 
[`baker.MetricsClient`](https://pkg.go.dev/github.com/AdRoll/baker#MetricsClient) in 
the component code

The two mechanisms following pull vs push approach respectively.
The `MetricsBag` should be returned by the `Stats` method along with the default
statistics and it will be collected 1 per second by the Baker runtime. Differently,
the `MetricsClient` can be requested from the Baker topology in the component
instantiation and it can be used in any part of the component code to report metrics.
If there is no particular requirements it is suggested to prefer the `MetricsBag`. 
Indeed, the pull approach permits the reduction the metrics overhead during the 
record processing.

Both `MetricsBag` and `MetricsClient` Metrics supports the most common metric types,
namely:
- `RawCounter`, a cumulative counter that can only increase.
- `DeltaCounter`, the total number of event occurrences in a unit time.
- `Gauge`, a snapshot of an event in the last unit time. 
- `Histogram`, a statistical distribution of a set of values in one unit 
of time.
- `Duration` or `Timing`, like an histogram but with time durations

Moreover, the `MetricsClient` is the direct interface to the remote metrics service 
used by the topology, e.g., *Datadog*, thus it supports some specific features such 
as **tags**.
Tags are a way of adding dimensions to telemetries so they can be filtered, 
aggregated, and compared in different visualizations.
Therefore, if a component requires to publish its metrics with a set of specific tags, the 
`MetricsClient` should use rather than `MetricsBag`.
However, it is suggested to use `MetricsClient` in the `Stats` method of the 
components to reduce contention and improve performance in the processing of the 
elements.

In summary, the go-to way is to implements custom statistics with a `MetricsBag`, but 
there are some situations in which it is preferable the `MetricsClient`, namely:
- the components need to publish the metrics using specific tags,
- the components cannot centralize the metric collections, e.g. the component has 
multiple goroutines.

#### MetricsBag Example

Let's say our filter needs to perform HTTP requests in order to decide whether a 
record should be discarded, we might want to keep track of the requests' durations 
in a histogram. In this case, we would probably record a slice of `time.Duration` in 
our filter and call 
[`AddTimings`](https://pkg.go.dev/github.com/AdRoll/baker#MetricsBag.AddTimings) on
the returned `MetricsBag`.

An important point is that Baker may call `Process` and `Stats` concurrently, from 
different goroutines so you must use proper locking on data structures that are
shared between these two methods. 

```go
func (f *MyFilter) Process(r Record, next func(Record)) {
    atomic.AddInt64(&myFilter.totalLines, 1)

    /* perform http request */
    f.mu.Lock() // keep track of its duration
    f.requestDurations = append(f.requestDurations, duration)
    f.mu.Unlock()

    if (/* filter logic*/) {
        // discard line
        atomic.AddInt64(&f.filteredLines, 1)
        return
    }
}

func (f *MyFilter) Stats() baker.FilterStats {
    // copy the slice of durations
    f.mu.Lock()
    requestDurations := f.requestDurations[:]
    f.mu.Unlock()

    bag := make(baker.MetricsBag)
    bag.AddTimings("myfilter_http_request_duration", requestDurations)
    return baker.FilterStats{
        NumProcessedLines: atomic.LoadInt64(&f.totalLines),
        NumFilteredLines: atomic.LoadInt64(&f.filteredLines),
        Metrics: bag,
    }
}
```

#### MetricsClient Example

Once a 
[`baker.MetricsClient`](https://pkg.go.dev/github.com/AdRoll/baker#MetricsClient) 
instance has been successfully created, it's made available to and used by a Baker 
pipeline to report metrics. During construction, components receive the 
[`MetricsClient`](https://pkg.go.dev/github.com/AdRoll/baker#MetricsClient) instance.
[`baker.MetricsClient`](https://pkg.go.dev/github.com/AdRoll/baker#MetricsClient) 
provides a set of methods to report the most common type of metric types (e.g. 
*gauges*, *counters* and *histograms*)

Taking the previous example of the `MetricsBag`, we now consider the situation in 
which the time duration that we want to collect, needs to be repported along with specific 
tags.

```go
func NewMyFilter(cfg baker.InputParams) (baker.Input, error) {
    return MyFilter{metrics: cfg.Metrics}
}

func (f *MyFilter) Process(r Record, next func(Record)) {
 /*... */
}

func (f *MyFilter) Stats() baker.FilterStats {
    f.mu.Lock()
    for _, duration := range f.requestDurations {
        f.metrics.DurationWithTags(
            "myfilter_http_request_duration", 
            duration, 
            []string{"tag1", "tag2"},
        )
    }
    f.mu.Unlock()

    return baker.FilterStats{
        NumProcessedLines: atomic.LoadInt64(&f.totalLines),
        NumFilteredLines: atomic.LoadInt64(&f.filteredLines),
    }
}
```