---
title: "CountAndTag"
weight: 11
date: 2022-07-05
---
{{% pageinfo color="primary" %}}

**Read the [API documentation &raquo;](https://pkg.go.dev/github.com/AdRoll/baker/filter#CountAndTag)**
{{% /pageinfo %}}

## Filter *CountAndTag*

### Overview
Publishes a metric that simply counts the number of records passing through, updating a metric of type counter.
In addition, the metric is also tagged with the value of a given, configured, field.
Records having an empty Field value are counted and tagged nonetheless, but they are tagged under the configured tag value: DefaultValue.

#### NOTE
A special attention should be given to the Field used to tag records.
For example, high-cardinality tags might cause performance degradation during tag ingestion and/or visualization and, depending on the metrics client you're using, could incur additional cost.
Something else to keep in mind is that not all values might be valid for the metrics client/system you're using. This filter does not try in any way to validate those.

Finally, it's good to keep in mind that metrics, like other means of observability (logs, tracing, etc.), are provided as best effort and should not influence the program outcome.
As such, it's important to have strong guarantees about the set of possible values for the configured Field, or else it could be necessary to perform some filtering prior to place this filter in your pipeline.


### Configuration

Keys available in the `[filter.config]` section:

|Name|Type|Default|Required|Description|
|----|:--:|:-----:|:------:|-----------|
| Metric| string| ""| true| Name of the metric of type counter published by this filter|
| Field| string| ""| true| Field which value is used to to break down the metric by tag values|
| DefaultValue| string| ""| true| Default tag value to use when the value of the configured field is empty|

