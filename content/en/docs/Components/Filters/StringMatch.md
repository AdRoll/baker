---
title: "StringMatch"
weight: 15
date: 2020-12-03
---
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

