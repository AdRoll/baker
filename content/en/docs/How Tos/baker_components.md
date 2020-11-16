---
title: "Create baker.Components"
date: 2020-10-29
weight: 500
description: >
  baker.Components is the main object used to create a Baker topology
---

To create a Topology, Baker requires 2 elements:

* `baker.Components` describes the list of components Baker can use in topologies
* a TOML configuration that specifically describes a single topology, using components from 1)

The next paragraphs gives you a high level overview of each section of `baker.Components`.

To get a deeper understanding, read the
[full API reference for `baker.Components`](https://pkg.go.dev/github.com/AdRoll/baker#Components).

## Inputs, Filters, Outputs and Uploads

These fields contain the list of components that are available to the topology.

The [TOML configuration file](/docs/core-concepts/toml/) must specify components that are
present in these lists.

All components already available to Baker or custom components can be set here.

The following is an example of `baker.Components` configuration where:

* **inputs** and **uploads** are those already included into Baker
* only a custom **filter** is set
* a custom **output** is added to the outputs included into Baker

```go
import (
	"github.com/AdRoll/baker"
	"github.com/AdRoll/baker/input"
	"github.com/AdRoll/baker/output"
	"github.com/AdRoll/baker/upload"
)

comp := baker.Components{
    Inputs:        input.All,
    Filters:       []baker.FilterDesc{MyCustomFilterDesc},
	Outputs:       append(output.All, MyCustomOutputDesc...),
	Uploads:       upload.All,
}
```

## Metrics

The list of available metrics backends.

This list can contain a metric backend already included into Baker or a custom implementation
of the `baker.MetricsClient` interface.

For details about metrics, [see the dedicated page](/docs/core-concepts/metrics).

## User

This field contains a list of user-defined configurations structures that are not strictly
useful to Baker but that users can add to Baker TOML file and use for other purposes.

To learn more about this topic, read the
[dedicated section](/docs/core-concepts/toml/#user-defined-configurations) in the Pipeline
configuration page.

## ShardingFuncs

This field holds a dictionary associating field indices to hash functions. When sharding
is enabled, these hash functions are used to determine which shard a record is sent to.

## Validate

`Validate` is the function used to validate a record. It is called for each processed record
unless `null` or when `[general.dont_validate_fields]` configuration is set to `true`.

Regardless of the TOML configuration, the function is passed to all components that can use
it at their will.

## CreateRecord

`CreateRecord` is the function that creates a new record. If not set, a default function is
used that creates a `LogLine` with the **comma** field separator.

The function is used internally by Baker to create new records every time a new byte buffer enters
the filter chain.

The function is also passed to components that can use it to create new records while processing.

## FieldByName

`FieldByName` returns a field index from its name.

The function is mainly used by the components (that receive it during setup) to retrieve the
index of a field they need for filtering or processing, but it is also used internally by
Baker when sending fields to the output (when at least one field is selected in the output
TOML configuration).

## FieldName

`FieldName` returns a field name from its index.

The function is passed to components that can use it internally.
