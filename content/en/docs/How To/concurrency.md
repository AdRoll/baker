---
title: "Tuning concurrency"
date: 2020-11-02
weight: 700
---

Baker allows to tune concurrency at various levels of a pipeline:

* input: Baker configuration doesn't expose knobs to tune input concurrency as it highly depends
on the input source and how the input is implemented
* filters: Baker runs N concurrent filter chains
* output: Baker runs M concurrent outputs

By default then, Baker processes records concurrently, without any guaranteed order.  
However, if you need to maintain the order of the records through the whole pipeline, it is still
possible by disabling concurrency ([see below for details](#guarantee-records-order)).

### Filter chain concurrency

The filter chain is a synchronous list of filters that are applied in the order in which they are
listed in the topology [TOML configuration file](/docs/core-concepts/toml/).

By default, though, Baker executes multiple concurrent filter chains (the default value is 16)

Filterchain concurrency can be set defining the `procs` key in the `[filterchain]` section:

```toml
[filterchain]
procs=16
```

Setting the value to **procs=1** disables the filter chain concurrency.

### Concurrent output

The output concurrency can be set defining the `procs` key in the `[output]` section:

```toml
[output]
procs=32
```

The default value is **32**.  
To disable concurrency, set **procs=1**.

#### Output concurrency support

For outputs that don't support concurrency, `procs=1` must be used to avoid corrupted output or
lost data.

Refer to the output documentation to know if it supports concurrent processing.

{{% alert color="info" %}}
We'll soon add a new function to the output to declare whether it supports concurrency,
and Baker will return an error if `procs>1` is used with an output that doesn't support it.
{{% /alert %}}

### Guarantee Records order

Although it's not the primary goal of Baker, it is still possible to disable concurrency and thus
guarantee records ordering from input to output.

To do so, add both `procs=1` for output and filterchain, disabling concurrent processing for
those components.
