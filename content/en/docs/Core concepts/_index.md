---
title: "Core concepts"
linkTitle: "Core concepts"
weight: 200
date: 2020-10-29
description: >
  Baker core concepts
---

## Pipeline

A pipeline (a.k.a. Topology) is the configured set of operations that Baker performs during
its execution.

It is defined by:

* One input component, defining where to fetch records from
* Zero or more filters, applied sequentially, which together define the **filter chain**. A filter is
a function that processes record (modifying fields, discarding or creating records).
* One output component, defining where to send the filtered records to (and which fields)
* One optional upload component, defining where to send files produced by the output component

Notice that there are two main usage scenarios for Baker, batch or daemon processing, that depend on
the input component behavior:

* **Batch processing**: In this case, Baker goes through all the records that are fed
by the input component, processes them as quickly as possible, and exits when the input component
ends its job.
* **Daemon**: in this case, the input component never exits and thus also Baker, that keeps waiting
for incoming records from the input (e.g.: Kinesis), processes them and sends them to the output.

Also read [Pipeline configuration](/docs/how-to/pipeline_configuration/)

## Record and LogLine

Baker processes "records". A `Record` is an interface that provides an abstraction over a record
of flattened data, where columns of fields are indexed through integers.

Baker currently provides a single implementation of Record, called `LogLine` (
[API reference](https://pkg.go.dev/github.com/AdRoll/baker#LogLine)).

If `LogLine` doesn't fit your needs, you can [customize it](/docs/how-to/record_and_logline/)
or [implement your version of the Record](/docs/how-to/custom_record/).

## Components

To process records, Baker uses up to 4 component types, each one with a different job:

* **Input** reads the input records (as raw data) and sends them to Baker
* Baker parses the records from the raw bytes received by the input and sends them through
the filter chain, an ordered list of **Filter** components that can modify, drop or create Records
* At the end of the filter chain, the records are sent to the **Output** component, whose job is
to save them somewhere.
* An optional **Upload** component receives the files produced by the Output and upload them to
their final destination.

Read our How-to guides to know how to:

* [create an Input component](/docs/how-tos/create_input/)
* [create a Filter component](/docs/how-tos/create_filter/)
* [create an Output component](/docs/how-tos/create_output/)
* [create an Upload component](/docs/how-tos/create_upload/)

## Metrics

During execution, Baker collects different kind of performance data points:

 * general pipeline metrics such as the total number of records processed and records per seconds.
 * component-specific metrics: files written per second, discarded records (by a filter), errors, etc.
 * Go runtime metrics: mallocs, frees, garbage collections and so on.

If enabled, Baker collects all these metrics and forwards them to a metrics client.

Metrics export is set up in Baker topology TOML files, [see how to configure it](/docs/how-to/metrics/).

Baker also prints general metrics once per second on standard output, in single-line format. Read more 
about it [here](/docs/how-tos/read_stats/).

## Sharding

Baker supports partitioning the records it processes into smaller subsets each
of which is forwarded to an output shard: _divide and conquer_.

When sharding is enabled, the shards, which are just multiple instances of the
same output component, run concurrently. Each of them only gets to process a 
specific subset of records, based on a the value a specific field has. This
horizontal partioning allows to get the most of the resources at your disposal,
since you can perform more work at the same time.

[Read more about sharding and how to configure it](/docs/how-to/sharding/)
