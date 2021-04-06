---
title: "MetadataUrl"
weight: 18
date: 2021-04-06
---
{{% pageinfo color="primary" %}}

**Read the [API documentation &raquo;](https://pkg.go.dev/github.com/AdRoll/baker/filter#MetadataUrl)**
{{% /pageinfo %}}

## Filter *MetadataUrl*

### Overview

This filter looks for 'url' in records metadata and copies it into a field of your choice, see DstField.
If it doesn't find the 'url' in the metadata, this filter clear DstField.

If you wish to discard records without the 'url' metadata, you can add the NotNull filter after this one in your topology.


### Configuration

Keys available in the `[filter.config]` section:

|Name|Type|Default|Required|Description|
|----|:--:|:-----:|:------:|-----------|
| DstField| string| ""| true| Name of the field into to write the url to (or to clear if there's no url)|

