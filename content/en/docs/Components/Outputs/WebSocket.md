---
title: "WebSocket"
weight: 25
date: 2020-12-14
---
{{% pageinfo color="primary" %}}

**Read the [API documentation &raquo;](https://pkg.go.dev/github.com/AdRoll/baker/output#WebSocket)**
{{% /pageinfo %}}

## Output *WebSocket*

### Overview
This is a *non-raw* output, it doesn't receive whole records. Instead it receives a list of fields for each record (`output.fields` in TOML).


This output writes the filtered log lines into any connected WebSocket client.  



### Configuration
No configuration available