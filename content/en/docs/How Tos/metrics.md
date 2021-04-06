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

## Implementing a custom MetricsClient

The [metrics example](https://github.com/AdRoll/baker/tree/main/examples/metrics) shows an
example implementation of
[`baker.MetricsClient`](https://pkg.go.dev/github.com/AdRoll/baker#MetricsClient)
and how to register it within Baker so that it can be selected in the
`[metrics.name]` TOML section.

In order to be selected from TOML, you should first register a 
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

## How to publish statistics from a component

All components need to implement a `Stats` method where they can expose metrics, called 
by Baker once per second. Each of the different component types support a set of predefined
metrics. In particular, the following counters are defined:

- [Input](https://pkg.go.dev/github.com/AdRoll/baker#InputStats):
    - `NumProcessedLines`, the total number of processed records since the component creation.
- [Filter](https://pkg.go.dev/github.com/AdRoll/baker#FilterStats):
   - `NumFilteredLines`, the number of filtered out (i.e. discarded) records.
- [Output](https://pkg.go.dev/github.com/AdRoll/baker#OutputStats):
    - `NumProcessedLines`, the total number of processed records since the component creation.
    - `NumErrorLines`, the number of records that have produced an error.
- [Upload](https://pkg.go.dev/github.com/AdRoll/baker#UploadStats):
    - `NumProcessedFiles`, the total number of processed files since the component creation.
    - `NumErrorFiles`, the number of files that have produced an error.

Due to historical reasons, these fields have the word _lines_ in them but they do 
mean the number of records.

The code example of 
[how-to-create-filter](https://getbaker.io/docs/how-tos/create_filter/#processing-records) 
shows how to correctly report metrics in a custom Filter.

### Report custom metrics

In addition to the record counters described above, components can report custom metrics
giving a more specific view about the component health or performance.
Baker supports two ways of exposing custom metrics:

- Returning a 
[`baker.MetricsBag`](https://pkg.go.dev/github.com/AdRoll/baker#MetricsBag)
instance from the `Stats` method.
- Directly using the 
[`baker.MetricsClient`](https://pkg.go.dev/github.com/AdRoll/baker#MetricsClient) in 
the component code.

The two mechanisms follow the pull vs push approaches, respectively.

The `MetricsBag` should be returned by the `Stats` method along with the default
statistics and is collected once per second by Baker. 

The `MetricsClient` instance configured by your topology is passed to your 
component during instantiation, in the `ComponentsParam` structure. You can copy it 
so as to use it at any point in your component code since well-behaved `MetricsClient` 
implementations are safe for concurrent use by multiple goroutines.

Both `MetricsBag` and `MetricsClient` Metrics support the most common metric types,
namely:
- `RawCounter`: a cumulative counter that can only increase.
- `DeltaCounter`: the total number of event occurrences in a unit time.
- `Gauge`: a snapshot of an event in the last unit time.
- `Histogram`: statistical distribution of a set of values in one unit of time.
- `Duration` or `Timing`: like a histogram but with time durations.

If there is no particular requirement the `MetricsBag` approach is preferred. Indeed, 
the `MetricsBag` pull approach is simpler and it better integrates with the 
other default metric gathering. 
In addition to that, the `MetricsClient` requires special attention to avoid too
many calls to the client during record processing.
Indeed, the `MetricsClient` is a low-level and direct way to communicate with your 
metrics system, thus it permits extra flexibility, but it skips the additional 
aggregation/buffering layer of the `MetricsBag`.

Moreover, the `MetricsClient` supports also the runtime tagging of the published metrics.
Tags (or labels) are additional dimensions on metrics and allow them to be filtered, 
aggregated, and compared in different visualizations.
Therefore, if tags are only known at runtime and you need to add dynamic tags to your metrics, 
the `MetricsClient` should be used rather than `MetricsBag` (see the 
[MetricsClient example](/docs/how-tos/metrics/#metricsclient-example)).

In summary, the go-to way is to implement custom statistics with a `MetricsBag`, but 
there are some situations where a `MetricsClient` is preferred, for example:
- Your component needs to publish the metrics using a set of tag that changes at runtime.
- You can't centralize metrics collection in the `Stats` method since the metrics
to expose are produced by different worker goroutines running inside your component 
multiple goroutines.

#### MetricsBag Example

Let's say our filter needs to perform HTTP requests in order to decide whether a 
record should be discarded, we might want to keep track of the requests' durations 
in an histogram. In this case, we would probably record a slice of `time.Duration` in 
our filter and call 
[`AddTimings`](https://pkg.go.dev/github.com/AdRoll/baker#MetricsBag.AddTimings) on
the returned `MetricsBag`.

An important point is that Baker may call `Process` and `Stats` from different goroutines
so access to the shared data must be properly synchronized (atomic, lock, channel, etc.)

```go
func (f *MyFilter) Process(r Record, next func(Record)) {
    // Perform http request

    f.mu.Lock() // keep track of its duration
    f.requestDurations = append(f.requestDurations, duration)
    f.mu.Unlock()

    if (/* filter logic*/) {
        // Discard record
        atomic.AddInt64(&f.filteredLines, 1)
        return
    }
}

func (f *MyFilter) Stats() baker.FilterStats {
    // Copy the slice of durations
    f.mu.Lock()
    requestDurations := f.requestDurations[:]
    f.mu.Unlock()

    bag := make(baker.MetricsBag)
    bag.AddTimings("myfilter_http_request_duration", requestDurations)
    return baker.FilterStats{
        NumFilteredLines: atomic.LoadInt64(&f.filteredLines),
        Metrics: bag,
    }
}
```

#### MetricsClient Example

Once a 
[`baker.MetricsClient`](https://pkg.go.dev/github.com/AdRoll/baker#MetricsClient) 
instance has been successfully created, it's made available to and can be used by a Baker 
pipeline to report metrics. During construction, components receive the 
[`MetricsClient`](https://pkg.go.dev/github.com/AdRoll/baker#MetricsClient) instance.
[`baker.MetricsClient`](https://pkg.go.dev/github.com/AdRoll/baker#MetricsClient) 
provides a set of methods to report the most common type of metric types (*gauges*,
*counters* and *histograms*)

Let's now consider we want to publish additional tags alongside the duration slice of our previous `MetricsBag` example.

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
        NumFilteredLines: atomic.LoadInt64(&f.filteredLines),
    }
}
```

Here, we chose to use the `MetricsClient` in the `Stats` method to limit the number of
calls to the client. Indeed, since `Stats` gets called once per second, the `MetricsClient` is 
also called once per second. We could have directly called `DurationWithTags` from the 
`Process` method but we would have probably - depending on the frequency of incoming 
records - called it way more often, and this could have potentially introduce some performance 
degradation. 
