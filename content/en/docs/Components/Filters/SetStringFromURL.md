---
title: "SetStringFromURL"
weight: 14
date: 2020-12-03
---
## Filter *SetStringFromURL*

### Overview
Extract some strings from metadata url and sets a field with it.  


### Configuration

Keys available in the `[filter.config]` section:

|Name|Type|Default|Required|Description|
|----|:--:|:-----:|:------:|-----------|
| Field| string| ""| true| Name of the field to set to|
| Strings| array of strings| []| true| Strings to look for in the URL. Discard records not containing any of them.|

