---
title: "Hash"
weight: 14
date: 2021-03-10
---
{{% pageinfo color="primary" %}}

**Read the [API documentation &raquo;](https://pkg.go.dev/github.com/AdRoll/baker/filter#Hash)**
{{% /pageinfo %}}

## Filter *Hash*

### Overview
This filter hashes a field using a specified hash function and writes the value 
to another (or the same) field. In order to have control over the set of characters
present, the hashed value can optionally be encoded.
	
	
Supported hash functions:
 - md5
 - sha256

Supported encodings:
- hex (hexadecimal encoding)


### Configuration

Keys available in the `[filter.config]` section:

|Name|Type|Default|Required|Description|
|----|:--:|:-----:|:------:|-----------|
| SrcField| string| ""| true| Name of the field to hash|
| DstField| string| ""| true| Name of the field to write the result to|
| Function| string| ""| true| Name of the hash function to use|
| Encoding| string| ""| false| Name of the encoding function to use|

