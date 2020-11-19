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

It is configured in a TOML file and is defined by:

* One input component, determining where to fetch records from
* Zero or more filters, applied sequentially, which together compose the **filter chain**. A filter is
a function that processes record: it can modify fields, discard records or create additional ones
* One output component, specifying where to send the records that made it so far
* One optional upload component, that can be added if the output creates files that need to be uploaded to
  a remote destination

Notice that there are two main usage scenarios for Baker, batch or daemon processing, that depend on
the input component behavior:

* **Batch processing**: In this case, Baker goes through all the records that are fed
by the input component, processes them as quickly as possible, and exits when the input component
ends its job.
* **Daemon**: in this case, the input component never exits and thus also Baker, that keeps waiting
for incoming records from the input (e.g.: Kinesis), processes them and sends them to the output.

Also read [Pipeline configuration](/docs/how-tos/pipeline_configuration/)

## Record and LogLine

Baker processes "records". A `Record` is an interface that provides an abstraction over a record
of flattened data, where columns of fields are indexed through integers.

Baker currently provides a single implementation of Record, called `LogLine` (
[API reference](https://pkg.go.dev/github.com/AdRoll/baker#LogLine)).

If `LogLine` doesn't fit your needs, you can [customize it](/docs/how-tos/record_and_logline/)
or [implement your version of the Record](/docs/how-tos/custom_record/).

## Components

To process records, Baker uses up to 4 component types, each one with a different job:

* **Input** reads blobs of data representing serialized records and sends them to Baker.
* Baker then parses the raw bytes, creates records from them and sends them through
the filter chain, an ordered list of **Filter** components that can modify, drop or create 
Records.
* At the end of the filter chain, records are sent to the **Output** component. There are 
2 types of output components. Raw outputs receive serialized records while non-raw outputs 
just receive a set of fields. Whatever its type, the output most certainly writes records
on disk or to an external service.
* In case the output saves files to disk, an optional **Upload** component can upload 
these files to a remote destination, such as Amazon S3 for example.

Read our How-to guides to know how to:

* [create an Input component](/docs/how-tos/create_input/)
* [create a Filter component](/docs/how-tos/create_filter/)
* [create an Output component](/docs/how-tos/create_output/)
* [create an Upload component](/docs/how-tos/create_upload/)

## Metrics

During execution, Baker collects different kind of performance data points:

 * General pipeline metrics such as the total number of records processed and records per seconds.
 * Component-specific metrics: files written per second, discarded records (by a filter), errors, etc.
 * Go runtime metrics: mallocs, frees, garbage collections and so on.

If enabled, Baker collects all these metrics and publishes them to a monitoring solution, such as Datadog
or Prometheus.

Metrics export is configured in Baker topology TOML files, [see how to configure it](/docs/how-tos/metrics/).

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

[Read more about sharding and how to configure it](/docs/how-tos/sharding/)
