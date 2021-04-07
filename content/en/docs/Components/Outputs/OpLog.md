---
title: "OpLog"
weight: 32
date: 2021-04-06
---
{{% pageinfo color="primary" %}}

**Read the [API documentation &raquo;](https://pkg.go.dev/github.com/AdRoll/baker/output#OpLog)**
{{% /pageinfo %}}

## Output *OpLog*

### Overview
This is a *non-raw* output, it doesn't receive whole records. Instead it receives a list of fields for each record (`output.fields` in TOML).


This output writes the filtered log lines into the current baker log, purely for development purpose.


### Configuration
No configuration available