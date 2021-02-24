---
title: "MetadataLastModified"
weight: 10
date: 2020-02-24
---

{{% pageinfo color="primary" %}}

**Read the [API documentation &raquo;](https://pkg.go.dev/github.com/AdRoll/baker/filter#MetadataLastModified)**
{{% /pageinfo %}}

## Filter _MetadataLastModified_

### Overview

Extract the last modified timestamp from the record Metadata and write it on a selected field.

### Configuration

Keys available in the `[filter.config]` section:

| Name     |  Type  | Default | Required | Description                               |
| -------- | :----: | :-----: | :------: | ----------------------------------------- |
| DstField | string |         |   true   | field name into which write the timestamp |
