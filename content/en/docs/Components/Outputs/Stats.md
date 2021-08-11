---
title: "Stats"
weight: 34
date: 2021-08-11
---
{{% pageinfo color="primary" %}}

**Read the [API documentation &raquo;](https://pkg.go.dev/github.com/AdRoll/baker/output#Stats)**
{{% /pageinfo %}}

## Output *Stats*

### Overview
This is a *raw* output, for each record it receives a buffer containing the serialized record, plus a list holding a set of fields (`output.fields` in TOML).


Compute various distributions of the records it receives and dumps that to CSV. It computes the distribution of record by size and the distribution of the values of certain fields


### Configuration

Keys available in the `[output.config]` section:

|Name|Type|Default|Required|Description|
|----|:--:|:-----:|:------:|-----------|
| CountEmptyFields| bool| false| false| Whether fields with empty values are counted or not|
| CSVPath| string| "stats.csv"| false| Path of the CSV file to create|
| TimestampField| string| ""| true| Name of a field containing a POSIX timestamp (in seconds) used to build the times stats|

