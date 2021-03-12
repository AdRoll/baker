---
title: "ExpandJSON"
weight: 12
date: 2021-03-12
---
{{% pageinfo color="primary" %}}

**Read the [API documentation &raquo;](https://pkg.go.dev/github.com/AdRoll/baker/filter#ExpandJSON)**
{{% /pageinfo %}}

## Filter *ExpandJSON*

### Overview

ExpandJSON extracts values from a JSON formatted record field and writes them into other fields of the same record.
It supports [JMESPath](https://jmespath.org/tutorial.html) to select the values to copy inside the JSON.

### Example

A possible filter configuration is:

	[[filter]]
	name="ExpandJSON"
		[filter.config]
		Source = "json_data"
		[filter.config.Fields]
		jfield1  = "field1"
		jfield2  = "field2"
		
In this example, the filter extracts values of the `jfield1` and `jfield2` keys of the JSON 
object present in field `json_data`of the record. Then, the values of that keys will be written into the field 
`field1` and `field2` of the same record.


### Configuration

Keys available in the `[filter.config]` section:

|Name|Type|Default|Required|Description|
|----|:--:|:-----:|:------:|-----------|
| Source| string| ""| true| record field that contains the json|
| Fields| map of strings to strings| | true| <JMESPath -> record field> map, the rest will be ignored|
| TrueFalseValues| array of strings| ["true", "false"]| false| bind the json boolean values to correstponding strings|

