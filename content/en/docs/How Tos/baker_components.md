---
title: "Create baker.Components"
date: 2020-10-29
weight: 500
description: >
  baker.Components is the main object used to create a Baker topology
---

The [`baker.Components`](https://pkg.go.dev/github.com/AdRoll/baker#Components) struct
lists all the components available to Baker when defining topologies.


Hence, to create a topology, Baker requires:

* an instance of [`baker.Components`](https://pkg.go.dev/github.com/AdRoll/baker#Components)
* a TOML configuration file describing the topology we want to run

```go
func main() {
	comp := baker.Component {
		// ...
	}

	f, _ := os.Open("/path/to/topology.go")
	cfg, _ := baker.NewConfigFromToml(f, components)

_ = baker.Main(cfg)
}
```

The next paragraphs give you a high level overview of each field of the 
[`baker.Components`](https://pkg.go.dev/github.com/AdRoll/baker#Components) struct.


## Inputs, Filters, Outputs and Uploads

These fields list the components that are available to topologies. All components present
in `baker.Components` can be used in the [TOML configuration file](/docs/core-concepts/toml/).

The following is an example of `baker.Components` where:

* we use all **inputs** and **uploads** provided in Baker repository
* only a single **filter** is set, a custom one we declared ourselves
* all Baker **outputs** are added in addition our own custom output

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

	// Other fields now shown here.
}
```

## Metrics

`Metrics` lists the metrics clients available when creating topologies.

```go
import (
	"github.com/AdRoll/baker"
	"github.com/AdRoll/baker/metrics"
)

comp := baker.Components{
    Metrics: metrics.All,

	// Other fields now shown here.
}
```


This list can contain a metric backend already included into Baker or a custom implementation
of the `baker.MetricsClient` interface.

For more, see the page [dedicated to metrics](/docs/core-concepts/metrics).

## User

```go
import "github.com/AdRoll/baker"

comp := baker.Components{
	User:    []baker.UserDesc{ /* list of user-specific structs */},

	// Other fields now shown here.
}
```

Baker users might want to use Baker TOML files to store application-specific configuration.
The `User` field lists user-defined configurations structures which aren't strictly
useful to Baker. 

To learn more about this topic, read the
[dedicated section](/docs/core-concepts/toml/#user-defined-configurations) in the Pipeline
configuration page.

## ShardingFuncs

```go
import "github.com/AdRoll/baker"

shardingFuncs := make(map[baker.FieldIndex]baker.ShardingFunc)

comp := baker.Components{
	ShardingFuncs: shardingFuncs,

	// Other fields now shown here.
}
```

`ShardingFuncs` holds a dictionary associating field indexes to hash functions. When sharding
is enabled, these hash functions are used to determine which shard a record is sent to.

## Validate

```go
import "github.com/AdRoll/baker"

func validate(baker.Record) (bool, baker.FieldIndex) {
	// ...
}

comp := baker.Components{
	Validate: validate,

	// Other fields now shown here.
}
```

`Validate` is the function used to validate a record. It is called for each processed record
unless `nil` or when `dont_validate_fields` is set to `true` in TOML's `[general]` section.

Regardless of the `dont_validate_fields` value, the `Validate` function is made accessible
to all components so that they can use it at their will.

## CreateRecord

```go
import "github.com/AdRoll/baker"

func create() baker.Record {
	// ...
}

comp := baker.Components{
	CreateRecord: create,

	// Other fields now shown here.
}
```

`CreateRecord` is a factory function returning new `Record` instances. If not set, a default function is
used that creates a `LogLine` with the **comma** field separator.

The function is used internally by Baker each time a new Record must be created. This
happens when blobs of raw serialized data, provided by the `Input` component, are parsed.

The function is also available for components needing to create new records.

## FieldByName

```go
import "github.com/AdRoll/baker"

func fieldByName(name string) (baker.FieldIndex, bool) {
	// ...
}

comp := baker.Components{
	FieldByName: fieldByName,

	// Other fields now shown here.
}
```

`FieldByName` returns the index of a field given its name.

Internally Baker refers to fields by their indexes, but it's simpler for users to refer to fields
with their names. This function exists to convert a field name to its index, it also controls
if the name is valid. 

The function is mainly used by the components (that receive it during setup) to retrieve the
index of a field they need for filtering or processing, but it is also used internally by
Baker when sending fields to the output (when at least one field is selected in the output
TOML configuration).

## FieldName

```go
import "github.com/AdRoll/baker"

func fieldName(idx baker.FieldIndex) string {
	// ...
}

comp := baker.Components{
	FieldName: fieldName,

	// Other fields now shown here.
}
```
`FieldName` returns a field name from its index.

The function is provided for components to use in case they need it.
