---
title: "Hash"
weight: 10
date: 2021-02-24
---

{{% pageinfo color="primary" %}}

**Read the [API documentation &raquo;](https://pkg.go.dev/github.com/AdRoll/baker/filter#Hash)**
{{% /pageinfo %}}

## Filter _Hash_

### Overview

Hash a field using a specified hash function and write the value on another (or the same) field.
Supported hash functions are:

- [MD5](https://en.wikipedia.org/wiki/MD5)
- [SHA256](https://en.wikipedia.org/wiki/SHA-2)

The filter also supports the [hexadecimal encoding](<https://en.wikipedia.org/wiki/Hexadecimal#Base16_(transfer_encoding)>) of the hash.

### Configuration

Keys available in the `[filter.config]` section:

| Name     |  Type  | Default | Required | Description                                                            |
| -------- | :----: | :-----: | :------: | ---------------------------------------------------------------------- |
| SrcField | string |         |   true   | field name to hash                                                     |
| DstField | string |         |   true   | field name into which write the result                                 |
| Function | string |         |   true   | name of the hash function to use, possible values are: `md5`, `sha256` |
| Encoding | string |  `""`   |  false   | name of the encoding function, possible value is: `hex`               |
