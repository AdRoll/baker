---
title: "URLParam"
weight: 31
date: 2022-07-05
---
{{% pageinfo color="primary" %}}

**Read the [API documentation &raquo;](https://pkg.go.dev/github.com/AdRoll/baker/filter#URLParam)**
{{% /pageinfo %}}

## Filter *URLParam*

### Overview

This filter extracts a query parameter (Param) from a source field (SrcField)
containing a URL and saves it into a destination field (DstField).

Error handling:
- If "SrcField" does not contain a valid URL an empty string will be stored in
DstField.
- If the query param "Param" is not present in the URL an empty string will be
stored in DstField.


### Configuration

Keys available in the `[filter.config]` section:

|Name|Type|Default|Required|Description|
|----|:--:|:-----:|:------:|-----------|
| SrcField| string| ""| true| field containing the url.|
| DstField| string| ""| true| field to save the extracted url param.|
| Param| string| ""| true| name of the url parameter to extract.|

