---
title: "Crypt"
weight: 12
date: 2022-07-05
---
{{% pageinfo color="primary" %}}

**Read the [API documentation &raquo;](https://pkg.go.dev/github.com/AdRoll/baker/filter#Crypt)**
{{% /pageinfo %}}

## Filter *Crypt*

### Overview

This filter encrypts or decrypts a field and writes the resulting value to another (or the same) field.

Supported algorithms:
 - fernet

### Fernet configuration

 - **Key**: 256-bit key used to encrypt/decrypt the token.
 - **TTL**: optional duration (in seconds). When set, the key must have been signed at most TTL ago, or decryption will fail. Only applicable for decryption.


### Configuration

Keys available in the `[filter.config]` section:

|Name|Type|Default|Required|Description|
|----|:--:|:-----:|:------:|-----------|
| Algorithm| string| ""| true| Name of the algorithm to use for crypting/decrypting|
| Decrypt| bool| false| false| True for decrypting, false for encrypting|
| SrcField| string| ""| true| Name of the field to crypt/decrypt|
| DstField| string| ""| true| Name of the field to write the result to|
| AlgorithmConfig| map of strings to strings| | false| AlgorithmConf contains configurations required by the chosen algorithm|

