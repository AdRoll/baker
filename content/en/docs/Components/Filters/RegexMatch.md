---
title: "RegexMatch"
weight: 23
date: 2022-12-12
---
{{% pageinfo color="primary" %}}

**Read the [API documentation &raquo;](https://pkg.go.dev/github.com/AdRoll/baker/filter#RegexMatch)**
{{% /pageinfo %}}

## Filter *RegexMatch*

### Overview
Discard record which have one or more fields that do not match their corresponding regular expressions

### Configuration

Keys available in the `[filter.config]` section:

|Name|Type|Default|Required|Description|
|----|:--:|:-----:|:------:|-----------|
| Fields| array of strings| []| false| list of fields to match with the corresponding regular expression in Regexs|
| Regexs| array of strings| []| false| list of regular expression to match. Fields[0] must match Regexs[0], Fields[1] Regexs[1] and so on|
| InvertMatch| bool| false| false| invert the match outcome, so that records are discarded if one or more fields match their corresponding regular expression|

