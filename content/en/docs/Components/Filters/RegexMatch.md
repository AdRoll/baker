---
title: "RegexMatch"
weight: 11
date: 2020-11-24
---
## Filter *RegexMatch*

### Overview
Discard a record if one or more fields don't match the corresponding regular expressions

### Configuration

Keys available in the `[filter.config]` section:

|Name|Type|Default|Required|Description|
|----|:--:|:-----:|:------:|-----------|
| Fields| array of strings| []| false| list of fields to match with the corresponding regular expression in Regexs|
| Regexs| array of strings| []| false| list of regular expression to match. Fields[0] must match Regexs[0], Fields[1] Regexs[1] and so on|

