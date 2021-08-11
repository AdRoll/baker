---
title: "SetStringFromURL"
weight: 24
date: 2021-08-11
---
{{% pageinfo color="primary" %}}

**Read the [API documentation &raquo;](https://pkg.go.dev/github.com/AdRoll/baker/filter#SetStringFromURL)**
{{% /pageinfo %}}

## Filter *SetStringFromURL*

### Overview

This filter looks for a set of strings in the URL metadata and sets a field with the found string.
Discards the log lines if URL metadata doesn't contain any of the given strings.

**On Error:** the input record is discarded.


### Configuration

Keys available in the `[filter.config]` section:

|Name|Type|Default|Required|Description|
|----|:--:|:-----:|:------:|-----------|
| Field| string| ""| true| Name of the field to set to|
| Strings| array of strings| []| true| Strings to look for in the URL. Discard records not containing any of them.|

