---
title: "Sharding"
date: 2020-11-02
weight: 500
---

# TBD - these are just notes

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
