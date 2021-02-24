---
title: "ExpandJSON"
weight: 11
date: 2021-02-24
---

{{% pageinfo color="primary" %}}

**Read the [API documentation &raquo;](https://pkg.go.dev/github.com/AdRoll/baker/filter#ExpandJSON)**
{{% /pageinfo %}}

## Filter _ExpandJSON_

### Overview

The filter copy the values of a set of JSON keys to corresponding record fields. The filter support also [JMESpath](https://jmespath.org/tutorial.html) to select the values to copy inside the JSON.

It is suggested to not use the default `,` CSV delimiter to avoid a possible clash with the JSON format. Change CSV delimiter with the `field_separator` configuration of the `csv`. For instance:

```
[csv]
    field_separator=";"
```

### Configuration

Keys available in the `[filter.config]` section:

| Name            |       Type           |      Default      | Required | Description                                                |
| --------------- | :------------------: | :---------------: | :------: | ---------------------------------------------------------- |
| Source          |       string         |                   |   true   | field name that contains the JSON to parse                 |
| Fields          | map string of string |                   |   true   | map the JMESPath expressions to field names                |
| TrueFalseValues |   array of strings   | ["true", "false"] |  false   | bind the JSON boolean values to the correstponding strings |


## Example

A possible filter configuration is:

```
[[filter]]
name="ExpandJSON"
	[filter.config]
	Source = "json_data"

	[filter.config.Fields]
	jfield1  = "field1"
    jfield2  = "field2"
```

The filter will transform the following input in the corresponding output:

**Input:**

| field1 | field2 |              json_data                 |
| :----: | :----: | :------------------------------------: |
|        |        | `{jfield1:"value1", jfield2:"value2"}` |

**Output:**

| field1 | field2 |              json_data                 |
| :----: | :----: | :------------------------------------: |
| value1 | value2 | `{jfield1:"value1", jfield2:"value2"}` |
