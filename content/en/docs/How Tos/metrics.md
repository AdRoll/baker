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

All components need to implement a `Stats` method where they can expose metrics. 
Baker calls the `Stats` method of each component once per second. `Stats` returns
a predefined set of metrics (depending on the component type) and a 
[`baker.MetricsBag`](https://pkg.go.dev/github.com/AdRoll/baker#MetricsBag),
in which one can add other metrics (of arbitrary name and type).

Let's illustrate this with metrics exported by a filter via 
[`baker.FilterStats`](https://pkg.go.dev/github.com/AdRoll/baker#FilterStats):

```go
type FilterStats struct {
	NumProcessedLines int64
	NumFilteredLines  int64
	Metrics           MetricsBag
}
```

In this case `NumProcessedLines` should represent the **total** number of processed 
lines since the filter creation, while `NumFilteredLines` is the number of discarded 
(i.e filtered) records. Due to historical reasons these fields have the word
_lines_ in them but they do mean the number of records.

#### A practical example

Let's say our filter needs to perform HTTP requests in order to decide whether a record
should be discarded, we might want to keep track of the requests' durations in an histogram.
In this case, we would probably record a slice of `time.Duration` in our filter and call
[`AddTimings`](https://pkg.go.dev/github.com/AdRoll/baker#MetricsBag.AddTimings) on the
returned `MetricsBag`.

An important point is that Baker may call `Process` and `Stats` concurrently, from different
goroutines so you must use proper locking on data structures which are shared between the
these two methods. 

```go
func (f *myFilter) Process(r Record, next func(Record)) {
    atomic.AddInt64(&myFilter.totalLines, 1)

    /* perform http request and keep track of its duration
     * in i.requestDurations
     */

    if (/* filter logic*/) {
        // discard line
        atomic.AddInt64(&myFilter.filteredLines, 1)
        return
    }
}

func (i *myFilter) Stats() baker.FilterStats {
    i.mu.Lock()
    bag := make(baker.MetricsBag)    
    bag.AddTimings("myfilter_http_request_duration", i.requestDurations)
    i.mu.Unlock()

    return baker.FilterStats{
        NumProcessedLines: atomic.LoadInt64(&myFilter.totalLines),
        NumFilteredLines: atomic.LoadInt64(&myFilter.filteredLines),
        Metrics: bag,
    }
}
```

#### Configuring metrics in TOML

Baker configuration TOML files may have a `[metrics]` section dedicated to the 
configuration of a metrics client.

`[metrics.name]` specifies the metrics client to use, from the list of all registered [`baker.MetricsClient`](https://pkg.go.dev/github.com/AdRoll/baker#MetricsClient).
`[metrics.config]` specifies some configuration settings which are specific to the client you're using.

For example, this is what the `[metrics]` section would look like with the *Datadog* metrics client:

```toml
[metrics]
name="datadog"

    [metrics.config]
    host="localhost:8125"                  # address of the dogstatsd client to which send metrics to
    prefix="myapp.baker."                  # prefix for all exported metric names
    send_logs=true                         # whether we should log messages (as Dogstatd events) or not 
    tags=["env:prod", "region:eu-west-1"]  # extra tags to associate to all exported metrics 
```

#### Disabling metrics

If you don't want to publish any metrics, it's enough to not provide the `[metrics]` TOML section in
Baker configuration file.


#### Implementing a new metrics client

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

#### Metrics.Client interface

Once a [`baker.MetricsClient`](https://pkg.go.dev/github.com/AdRoll/baker#MetricsClient)
instance has been successfully created, it's made available to and used by
a Baker pipeline to report metrics. During construction, components receive the 
[`MetricsClient`](https://pkg.go.dev/github.com/AdRoll/baker#MetricsClient) instance.

[`baker.MetricsClient`](https://pkg.go.dev/github.com/AdRoll/baker#MetricsClient) 
supports the most common type of metric types: *gauges*, *counters* and *histograms*.

