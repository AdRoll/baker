---
title: "URLEscape"
weight: 30
date: 2022-07-05
---
{{% pageinfo color="primary" %}}

**Read the [API documentation &raquo;](https://pkg.go.dev/github.com/AdRoll/baker/filter#URLEscape)**
{{% /pageinfo %}}

## Filter *URLEscape*

### Overview
Escape/Unescape URL. Escaping always succeeds but unescaping may fail, in which case this filter clears the destination field.

### Configuration

Keys available in the `[filter.config]` section:

|Name|Type|Default|Required|Description|
|----|:--:|:-----:|:------:|-----------|
| SrcField| string| ""| true| Name of the field with the URL to escape/unescape|
| DstField| string| ""| true| Name of the field to write the escaped/unescaped URL to.|
| Unescape| bool| false| false| Unescape the field instead of escaping it.|

