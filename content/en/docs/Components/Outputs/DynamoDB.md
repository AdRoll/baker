---
title: "DynamoDB"
weight: 17
date: 2020-11-24
---
## Output *DynamoDB*

### Overview
This is a *non-raw* output, it doesn't receive whole records. Instead it receives a list of fields for each record (`output.fields` in TOML).


This output writes the filtered log lines to DynamoDB.  
 It must be
configured specifying the region, the table name, and the columns
to write.  

Columns are specified using the syntax "t:name" where "t"
is the type of the data, and "name" is the name of column.  
 Supported
types are: "n" - integers; "s" - strings.  

The first column (and field) must be the primary key.  



### Configuration

Keys available in the `[output.config]` section:

|Name|Type|Default|Required|Description|
|----|:--:|:-----:|:------:|-----------|
| Regions| array of strings| us-west-2| false| DynamoDB regions to connect to|
| Table| string| ""| true| Name of the table to modify|
| Columns| array of strings| []| false| Table columns that correspond to each of the fields being written|
| FlushInterval| duration| 1s| false| Interval at which flush the data to DynamoDB even if we have not reached 25 records|
| MaxWritesPerSec| int| 0| false| Maximum number of writes per second that DynamoDB can accept (0 for unlimited)|
| MaxBackoff| duration| 2m| false| Maximum retry/backoff time in case of errors before giving up|

