---
title: "NotNull"
weight: 15
date: 2021-03-03
---
{{% pageinfo color="primary" %}}

**Read the [API documentation &raquo;](https://pkg.go.dev/github.com/AdRoll/baker/filter#NotNull)**
{{% /pageinfo %}}

## Filter *NotNull*

### Overview
Discard the records having null (i.e empty) fields.


### Configuration

Keys available in the `[filter.config]` section:

|Name|Type|Default|Required|Description|
|----|:--:|:-----:|:------:|-----------|
| Fields| array of strings| []| true| Fields is the list of fields to check for null/empty values|
