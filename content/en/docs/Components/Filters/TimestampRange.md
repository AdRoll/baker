---
title: "TimestampRange"
weight: 22
date: 2021-03-03
---
{{% pageinfo color="primary" %}}

**Read the [API documentation &raquo;](https://pkg.go.dev/github.com/AdRoll/baker/filter#TimestampRange)**
{{% /pageinfo %}}

## Filter *TimestampRange*

### Overview
Discard records if the value of a field containing a timestamp is out of the given time range (i.e StartDateTime <= value < EndDateTime)

### Configuration

Keys available in the `[filter.config]` section:

|Name|Type|Default|Required|Description|
|----|:--:|:-----:|:------:|-----------|
| StartDatetime| string| "no bound"| true| Lower bound of the accepted time interval (inclusive, UTC) format:'2006-01-31 15:04:05'. Also accepts 'now'|
| EndDatetime| string| "no bound"| true| Upper bound of the accepted time interval (exclusive, UTC) format:'2006-01-31 15:04:05'. Also accepts 'now'|
| Field| string| ""| true| Name of the field containing the Unix EPOCH timestamp|
