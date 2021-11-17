---
title: "StringMatch"
weight: 26
date: 2021-11-17
---
{{% pageinfo color="primary" %}}

**Read the [API documentation &raquo;](https://pkg.go.dev/github.com/AdRoll/baker/filter#StringMatch)**
{{% /pageinfo %}}

## Filter *StringMatch*

### Overview
Discard records if a field matches any of the provided strings

### Configuration

Keys available in the `[filter.config]` section:

|Name|Type|Default|Required|Description|
|----|:--:|:-----:|:------:|-----------|
| Field| string| ""| true| name of the field which value is used for string comparison|
| Strings| array of strings| []| true| list of strings to match.|
| InvertMatch| bool| false| false| Invert the match outcome, so that records are discarded if they don't match any of the strings|

