---
title: "Export metrics"
date: 2020-11-03
weight: 400
description: >
  How to export metrics from Baker
---

During its execution, a Baker pipeline exports metrics about the Go runtime as
well as general metrics giving an high-level overview of the ongoing job.

More specific metrics are also exported on a per-component basis. To that effect, 
`baker.Input`, `baker.Filter`, `baker.Output` and `baker.Upload` all have a `Stats` 
method. `Stats` is called every second and the component is expected to return both
a predefined set of metrics and a [`baker.MetricsBag`](https://pkg.go.dev/github.com/AdRoll/baker#MetricsBag)
containing metrics of arbitrary name and types.

Let's illustrate this with metrics exported by a filter via 
[`baker.FilterStats`](https://pkg.go.dev/github.com/AdRoll/baker#FilterStats):

```go
type FilterStats struct {
	NumProcessedLines int64
	NumFilteredLines  int64
	Metrics           MetricsBag
}
```

In this case `NumProcessedLines` must represent the **total** number of processed 
lines since Baker started, and `NumFilteredLines` is the number of discarded 
(or filtered) records. Due to historical reasons these fields have the word
_lines_ in them but they do mean the number of records.

An important point is that `Stats` can be called from any goroutine so it must be
safe for concurrent use by multiple goroutines. 

```go
func (i *myFilter) Stats() baker.FilterStats {
    bag := make(baker.MetricsBag)
    bag.AddGauge("current_speed", float64(atomic.LoadInt64(&myFilter.speed)))

    return baker.FilterStats{
        NumProcessedLines: atomic.LoadInt64(&myFilter.totalLines),
        NumFilteredLines: atomic.LoadInt64(&myFilter.filteredLines),
        // Metrics could be let to its default value, nil, if not needed.
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

#### Disaling metrics export

To not export any metrics, it's enough to not provide the `[metrics]` section in
Baker configuration file.


#### Implementing a new metrics client

The [metrics
example](https://github.com/AdRoll/baker/tree/main/examples/metrics) shows an
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

