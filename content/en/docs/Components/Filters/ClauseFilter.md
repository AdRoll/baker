---
title: "ClauseFilter"
weight: 8
date: 2020-11-12
---
## Filter *ClauseFilter*

### Overview
This filter lets you set a boolean expression (in s-expression format) that will be matched against all records and dropped if they don't match the expression.

Check the filter file for documentation what the format looks like.



### Configuration

Keys available in the `[filter.config]` section:

|Name|Type|Default|Required|Description|
|:--:|:--:|:-----:|:------:|:---------:|
| Clause| string| ""| false| Boolean formula describing which events to let through. If empty, let everything through.|

