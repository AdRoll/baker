---
title: "RegexMatch"
weight: 22
date: 2021-11-17
---
{{% pageinfo color="primary" %}}

**Read the [API documentation &raquo;](https://pkg.go.dev/github.com/AdRoll/baker/filter#RegexMatch)**
{{% /pageinfo %}}

## Filter *RegexMatch*

### Overview
Discard a record if one or more fields don't match the corresponding regular expressions

### Configuration

Keys available in the `[filter.config]` section:

|Name|Type|Default|Required|Description|
|----|:--:|:-----:|:------:|-----------|
| Fields| array of strings| []| false| list of fields to match with the corresponding regular expression in Regexs|
| Regexs| array of strings| []| false| list of regular expression to match. Fields[0] must match Regexs[0], Fields[1] Regexs[1] and so on|

