---
title: "PartialClone"
weight: 21
date: 2021-08-11
---
{{% pageinfo color="primary" %}}

**Read the [API documentation &raquo;](https://pkg.go.dev/github.com/AdRoll/baker/filter#PartialClone)**
{{% /pageinfo %}}

## Filter *PartialClone*

### Overview
Copy a list of fields to a new record and process this new record, discarding the original one

### Configuration

Keys available in the `[filter.config]` section:

|Name|Type|Default|Required|Description|
|----|:--:|:-----:|:------:|-----------|
| Fields| array of strings| []| true| Fields that must be copied to the new line|

