---
title: "ExpandList"
weight: 14
date: 2021-04-06
---
{{% pageinfo color="primary" %}}

**Read the [API documentation &raquo;](https://pkg.go.dev/github.com/AdRoll/baker/filter#ExpandList)**
{{% /pageinfo %}}

## Filter *ExpandList*

### Overview

This filter splits a field using a configured separator and writes the resulting values to other fields of the same 
record. The mapping between the extracted values and the destination fields is configured with a TOML table. The elements 
of the list are, by default, separated with the `;` character, but it is configurable.

### Example

A possible filter configuration is:

	[[filter]]
	name="ExpandList"
		[filter.config]
		Source = "list_data"
		Separator = ";"
		[filter.config.Fields]
		0 = "field1"
		1 = "field2"
		
In this example, the filter extracts the first and the second element of the list present in the field 
`list_data`of the record. Then, the values of that keys will be written into the field 
`field1` and `field2` of the same record.


### Configuration

Keys available in the `[filter.config]` section:

|Name|Type|Default|Required|Description|
|----|:--:|:-----:|:------:|-----------|
| Source| string| ""| true| record field that contains the list|
| Fields| map of strings to strings| | true| <list index -> record field> map, the rest will be ignored|
| Separator| string| ";"| false| character separator of the list|

