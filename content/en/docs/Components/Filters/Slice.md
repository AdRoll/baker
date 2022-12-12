---
title: "Slice"
weight: 26
date: 2022-12-12
---
{{% pageinfo color="primary" %}}

**Read the [API documentation &raquo;](https://pkg.go.dev/github.com/AdRoll/baker/filter#Slice)**
{{% /pageinfo %}}

## Filter *Slice*

### Overview
Slices the source field value using start/end indexes and copies the value to the destination field.
If the start index is greater than the field length, Slice sets the destination to an empty string.
If the end index is greater than the field length, Slice considers the end index to be equal to the field length.
Note: Indexes are 0-based and are intended as number of bytes, thus not taking into account any encoding the values may have.

### Configuration

Keys available in the `[filter.config]` section:

|Name|Type|Default|Required|Description|
|----|:--:|:-----:|:------:|-----------|
| Src| string| ""| true| The source field to slice|
| Dst| string| ""| true| The destination field to save the sliced value to|
| StartIdx| int| 0| false| The index representing where the slicing starts|
| EndIdx| int| | false| The index representing where the slicind ends. Defaults to the last byte|

