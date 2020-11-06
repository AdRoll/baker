---
title: "Sharding"
date: 2020-10-29
weight: 410
description: >
  Sharding how-to
---

Sharding is enabled in a topology by specifying a `sharding` field in the TOML
`[output]` section. That field must be shardable though, in other words, a 
sharding function must be associated to that field. Let's see how that works.


#### The baker.Components structure

The `baker.Components` structure links elements that may appear in the 
configuration to the code eventually running when these elements are used
inside a topology.


#### Register sharding functions

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
the sharding function that implement the hashing of that field.


#### Putting it all together

The following is an example of an hypothetical record schema with 3 fields 
named `timestamp`, `city` and `country`. Let's say that we'd like to use 
`timestamp` and `country` for sharding but not `city`. This is how implementing
sharding for such a schema would look probably like:

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