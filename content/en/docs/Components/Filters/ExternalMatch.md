---
title: "ExternalMatch"
weight: 16
date: 2022-07-05
---
{{% pageinfo color="primary" %}}

**Read the [API documentation &raquo;](https://pkg.go.dev/github.com/AdRoll/baker/filter#ExternalMatch)**
{{% /pageinfo %}}

## Filter *ExternalMatch*

### Overview
Discards records which fields matches values read from a CSV, which is possibly periodically refreshed. CSV files can be compressed (gz or zstd) or not.

### Configuration

Keys available in the `[filter.config]` section:

|Name|Type|Default|Required|Description|
|----|:--:|:-----:|:------:|-----------|
| Region| string| "us-west-2"| false| AWS region to pass to S3 client (only for files with s3:// prefix)|
| Files| array of strings| []| true| URL(s) of CSV file(s) containing the strings to match (s3[n]:// or file://). If %s is present, it's replaced, at download time, with the result of calling time.Now().Format(DateTimeLayout).|
| DateTimeLayout| string| ""| false| Go date time string layout replacing %s in Files, evaluated just before downloading Files. See https://pkg.go.dev/time#Time.Format|
| TimeSubtract| duration| | false| Duration to subtract from time.Now() when evaluating DateTimeLayout. See https://pkg.go.dev/time#ParseDuration|
| RefreshEvery| duration| | false| Period at which Files are refreshed (downloaded again), if not set, Files are never refreshed|
| CSVColumn| int| 0| false| 0-based index of the CSV column containing the values to consider|
| FieldName| string| ""| true| Name of the record field to consider for the match|
| KeepOnMatch| bool| false| false| If true, keep records if field at FieldName matches any of the CSV values. If false, discard records if field matches any of the CSV values.|

