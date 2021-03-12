---
title: "FormatTime"
weight: 14
date: 2021-03-12
---
{{% pageinfo color="primary" %}}

**Read the [API documentation &raquo;](https://pkg.go.dev/github.com/AdRoll/baker/filter#FormatTime)**
{{% /pageinfo %}}

## Filter *FormatTime*

### Overview

This filter formats and converts date/time strings from one format to another. 
It requires the source and destination field names along with 2 format strings, the 
first one indicates how to parse the input field while the second how to format it.

The source time parsing can fail if the time value does not match the provided format.
In this situation the filter clears the destination field, thus the user can filter out 
those results with a __NotNull__ filter.

Most standard formats are supported out of the box and you can provide your own format 
string, see [Go time layout](https://pkg.go.dev/time#pkg-constants).

Supported time format are:
- `ANSIC` format: "Mon Jan _2 15:04:05 2006"
- `UnixDate` format: "Mon Jan _2 15:04:05 MST 2006"
- `RubyDate` format: "Mon Jan 02 15:04:05 -0700 2006"
- `RFC822` format: "02 Jan 06 15:04 MST"
- `RFC822Z` that is RFC822 with numeric zone, format: "02 Jan 06 15:04 -0700"
- `RFC850` format: "Monday, 02-Jan-06 15:04:05 MST"
- `RFC1123` format: "Mon, 02 Jan 2006 15:04:05 MST"
- `RFC1123Z` that is RFC1123 with numeric zone, format: "Mon, 02 Jan 2006 15:04:05 -0700"
- `RFC3339` format: "2006-01-02T15:04:05Z07:00"
- `RFC3339Nano` format: "2006-01-02T15:04:05.999999999Z07:00"
- `unix` unix epoch in seconds
- `unixms` unix epoch in milliseconds
- `unixns` unix epoch in nanoseconds


### Configuration

Keys available in the `[filter.config]` section:

|Name|Type|Default|Required|Description|
|----|:--:|:-----:|:------:|-----------|
| SrcField| string| ""| true| Field name of the input time|
| DstField| string| ""| true| Field name of the output time|
| SrcFormat| string| "UnixDate"| false| Format of the input time|
| DstFormat| string| "unixms"| false| Format of the output time|

