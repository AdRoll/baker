---
title: "Sharding"
date: 2020-11-03
weight: 400
description: >
  Output sharding
---

### Overview

Baker supports partitioning the records it processes into smaller subsets each
of which is forwarded to an output shard: _divide and conquer_.

When sharding is enabled, the shards, which are just multiple instances of the
same output component, run concurrently. Each of them only gets to process a 
specific subset of records, based on a the value a specific field has. This
horizontal partioning allows to get the most of the resources at your disposal,
since you can perform more work at the same time.


#### Limitations

Baker only supports sharding at the output level. Baker implements other 
strategies so that other types of components (input, filters and uploads) 
maximize the pipeline performance.

Also, keep in mind that not all tasks can be parallelized, so not all outputs
support sharding. So sharding is an intrinsic property that is only present on
some Output components, but not all of them.

Only a single field can be used for sharding.


#### How to enable sharding in a topology?

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


#### Hash functions

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
field are arbitraty long strings. 

It's however possible to use a non-optimal but best effort general hash function.
(we're planning to add this to Baker soon).

A hash function should of course be deterministic (i.e the same input should 
always give the same output).