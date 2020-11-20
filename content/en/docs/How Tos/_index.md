---
title: "How-Tos"
weight: 300
description: >
  How to start with Baker as a library.
---

To configure and run a Baker topology, 4 steps are required:

1. use a [TOML configuration file](/docs/core-concepts/toml/)
2. define a [baker.Components object](/docs/how-tos/baker_components/)
3. obtain a Baker configuration object calling
[`baker.NewConfigFromToml`](https://pkg.go.dev/github.com/AdRoll/baker#NewConfigFromToml)
4. run [`baker.Main`](https://pkg.go.dev/github.com/AdRoll/baker#Main)

The [example folder](https://github.com/AdRoll/baker/tree/main/examples) in the Baker
repositories contains many examples of implementing a Baker pipeline.

Start with the [basic example](https://github.com/AdRoll/baker/blob/main/examples/basic/main.go).
