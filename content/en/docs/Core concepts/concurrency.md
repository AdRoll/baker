---
title: "Tuning concurrency"
date: 2020-11-02
weight: 3
---

Baker filters and ouput can be configured with a high concurrency profile (input doesn't support
parallel processing).

This means that those components will process Records in parallel during the pipeline,
improving performances but adding a small cons: Records order isn't guaranteed anymore
(due to possible different speed processing between concurrent components).

{{% alert title="Default behaviour" color="primary" %}}
The default behaviour of Baker is to use concurrency for both filters and output.
{{% /alert %}}

### Concurrent filtering

The filters concurrency can be set defining the `procs` key in the `[filterchain]` section:

```toml
[filterchain]
procs=16
```

The default value is **16**.  
To disable concurrency, set **procs=1**.

### Concurrent output

The output concurrency can be set defining the `procs` key in the `[output]` section:

```toml
[output]
procs=32
```

The default value is **32**.  
To disable concurrency, set **procs=1**.

#### Output concurrency support

At the moment it is not possible to know whether an output supports concurrency. A good guess is to
see if the can support sharding (the `CanShard()` function), but the proper way is to know the
output, whether reading its doc or looking at the code.

If an output doesn't support concurrency, you should use `procs=1` to avoid corrupted output or
lost data.

We'll soon add a new function to the output to declare its support for concurrency, and Baker will
return an error if `procs>1` is used with an output that doesn't support it.

### Guarantee Records order

Although it's not the primary goal of Baker, it is still possible to disable concurrency and thus
guarantee that the order of Records coming from the input is maintained in the output.

To do so, add both `procs=1` for output and filterchain.
