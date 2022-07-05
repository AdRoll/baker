---
title: "Dedup"
weight: 13
date: 2022-07-05
---
{{% pageinfo color="primary" %}}

**Read the [API documentation &raquo;](https://pkg.go.dev/github.com/AdRoll/baker/filter#Dedup)**
{{% /pageinfo %}}

## Filter *Dedup*

### Overview

This filter removes duplicate records. A record is considered a duplicate, and is thus removed by this filter, 
if another record with the same values has already been _seen_. The comparison is performed on a 
user-provided list of fields (`Fields` setting).

**WARNING**: to remove duplicates, this filter stores one key per unique record in memory, this means 
that the overall memory grows linearly with the number of unique records in your data set. Depending 
on your data set, this might lead to OOM (i.e. out of memory) errors.


### Configuration

Keys available in the `[filter.config]` section:

|Name|Type|Default|Required|Description|
|----|:--:|:-----:|:------:|-----------|
| Fields| array of strings| []| true| fields to consider when comparing records|
| KeySeparator| string| "\x1e"| false| character separator used to build a key from the fields|

