---
title: "Nop"
weight: 35
date: 2022-12-12
---
{{% pageinfo color="primary" %}}

**Read the [API documentation &raquo;](https://pkg.go.dev/github.com/AdRoll/baker/output#Nop)**
{{% /pageinfo %}}

## Output *Nop*

### Overview
This is a *non-raw* output, it doesn't receive whole records. Instead it receives a list of fields for each record (`output.fields` in TOML).


No-operation output. This output simply drops all lines and does not write them anywhere.

### Configuration
No configuration available