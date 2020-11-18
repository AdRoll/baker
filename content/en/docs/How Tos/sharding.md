---
title: "Sharding setup"
date: 2020-10-29
weight: 780
---

## How to configure a topology with output sharding?

Sharding is enabled in the `[output]` section of the topology TOML 
configuration file, by indicating the name of the field used to partition
the records space.

In the following topology extract, we're using a sharded `Filewriter` output
and set the number of instances to 4 (i.e 4 shards). In our case, Baker is 
going to extract and hash the `name` field of each `Record` to determine which
of the 4 `Filewriter` instances a `Record` is sent to: 

```toml
[input]
...

[[filter]]
...

[output]
name="Filewriter"
sharding="name"
procs=4

    [output.config]
    ...
```

## Limitations

Baker only supports sharding at the output level. Baker implements other 
strategies so that other types of components (input, filters and uploads) 
maximize the pipeline performance.

Also, keep in mind that not all tasks can be parallelized, so not all outputs
support sharding. So sharding is an intrinsic property that is only present on
some Output components, but not all of them.

Only a single field can be used for sharding.

## Hash functions

The field selected for sharding must be "shardable": in other words, a sharding function (or
hash function) must be associated to that field.

Since the aim of sharding is to uniformly distribute the load of incoming 
records between multiple instances of an output component, a good hash function
should be **uniform**; in other words it should map as evenly as possible from 
the range of possible input values to the range of output values.

The range of output values is known, it is  `[0, MaxUint64]` since in Baker 
hashes are `uint64` values).

However the range of possible input values depends on the domain. That's where
having knowledge of that particular domain will help in designing a hash 
function, that both guarantees the uniformity of output values with respect to 
input values, and in terms of performance.

For example, if you know the sharded field is only made of integers from 0 to 
1000, the hash function would be implemented differently than if the values for that 
field are arbitrary long strings.

It's however possible to use a non-optimal but best effort general hash function.
(we're planning to add this to Baker soon).

A hash function should of course be deterministic (i.e the same input should 
always give the same output).

## Register sharding functions

The `baker.Components` structure links elements that may appear in the 
configuration, to the code eventually running when these elements are used
inside a topology.

Sharding functions that may be used in topologies are stored inside of 
the `ShardingFuncs` field of [`baker.Components`](https://pkg.go.dev/github.com/AdRoll/baker#Components).

```go
ShardingFuncs map[baker.FieldIndex]ShardingFunc
```

And a [`ShardingFunc`](https://pkg.go.dev/github.com/AdRoll/baker#ShardingFunc)
is a hash function that returns an `uint64` `for baker.Record`

```go
type ShardingFunc func(Record) uint64
```

Finally, filling `ShardingFuncs` is a matter of associating a shardable field to
the sharding function that implements the hashing of that field.


## Putting it all together

The following is an example of an hypothetical record schema with 3 fields 
named `timestamp`, `city` and `country`. Let's say that we'd like to use 
`timestamp` and `country` for sharding but not `city`. We're going to enable
sharding on these two fields, but note that only one of them can be chosen
for a given topology.

This is how implementing sharding for such a schema would look probably like:

```go
const (
    FieldTimestamp baker.FieldIndex = 0 // timestamp is unix epoch timestamp
    FieldCity baker.FieldIndex      = 1 // city name
    FieldCountry baker.FieldIndex   = 2 // 2 chars country code
)
```

This is an hypothetical function to hash records based on the `timestamp` field
which only contains integers:

```go
func hashTimestamp(r baker.Record) uint64 {
    // We know the timestamp is an integer, so we use that 
    // to efficiently compute a hash from it.
    buf := r.Get(FieldTimestamp)
    ts, _ := strconv.Atoi(string(buf))

    // Call super efficient integer hash function
    return hashInt(ts)
}
```

And this is how hashing records based on a 2-char `country` code field would 
look like:

```go
func hashCountry(r baker.Record) uint64 {
    // We know the country is made of 2 characters, so we use that 
    // fact to efficiently compute a hash from it.
    buf := r.Get(FieldCountry)
    country := buf[:2]

    // Call our super fast function that hashes 2 bytes.
    return hash2bytes(country)
}
```

You can find [here](https://github.com/AdRoll/baker/tree/main/examples/sharding)
a full working example illustrating sharding in Baker.
