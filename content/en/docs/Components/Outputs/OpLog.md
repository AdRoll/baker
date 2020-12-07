---
title: "OpLog"
weight: 22
date: 2020-12-03
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