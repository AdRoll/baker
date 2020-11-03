---
title: "baker.Components"
date: 2020-10-29
weight: 320
description: >
  baker.Components is the main object used to create a Baker topology
---

`baker.Components` is a struct that is used to configure the Baker topology, before running it.

It contains the available components and some functions used by Baker to deal with records.

Read the [full API reference](https://pkg.go.dev/github.com/AdRoll/baker#Components).

## Inputs, Filters, Outputs and Uploads

These fields contain the list of components that are available to the topology.

The [TOML configuration file](/docs/core-concepts/toml/) must specify components that are
present in these lists.

Both components already available to Baker or custom components can be set here.

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

The list of available metrics bakends.

As for the components explained in the previous paragraph, metrics can be imported from what
is included into Baker or can be a custom implementation of the `baker.MetricsClient` interface
(or a mix of the two options).

For details about metrics, [see the dedicated page](/docs/core-concepts/metrics).

## User

This field contains the list of user-defined configurations, that are configurations not strictly
used by the Baker topology but that a user can read from the Baker TOML configuration file and use
anywhere in the code.

To learn more about this topic, read the
[dedicated section](/docs/core-concepts/toml/#user-defined-configurations) in the Pipeline
configuration page.

## ShardingFuncs

This field is a dictionary of functions used to calculate how to shard records among the
available output goroutines (see [Tuning concurrency](/docs/core-concepts/concurrency) for
details about output concurrency).

The `[output.sharding]` configuration in the TOML file tells which field of the records must be
used to calculate the sharding.

The field name, transformed to `FieldIndex` thanks to the `FieldByName` function (see
[below](#fieldbyname)), is the index of the map that corresponds to a sharding function.

The sharding function receives a Record and returns a sharding value (`uint64`) that Baker
uses to send the Record to an output process.

{{% alert color="info" %}}
How the sharding value is calculated is up to the function, but it should try to have a linear
distribution of records (to evenly distribute the load among the output concurrent processes)
and, although not required, it should also use value of the field that has been configured in
`[output.sharding]`.
{{% /alert %}}

### Validate

`Validate` is the function used to validate a record. It is called for each processed record
unless `null` or when `[general.dont_validate_fields]` configuration is set to `true`.

Regardless of the TOML configuration, the function is passed to all components that can use
it at their will.

### CreateRecord

`CreateRecord` is the function that creates a new record. If not set, a default function is
used that creates a `LogLine` with the **comma** field separator.

The function is used internally by Baker to create new records every time a new byte buffer enters
the filter chain.

The function is also passed to components that can use it to create new records while processing.

### FieldByName

`FieldByName` returns a field index from its name.

The function is mainly used by the components (that receive it during setup) to retrieve the
index of a field they need for filtering or processing, but it is also used internally by
Baker when sending fields to the output (when at least one field is selected in the output
TOML configuration).

### FieldName

`FieldName` returns a field name from its index.

The function is passed to components that can use it internally.
