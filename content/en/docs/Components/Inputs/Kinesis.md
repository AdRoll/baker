---
title: "Kinesis"
weight: 3
date: 2020-12-03
---
{{% pageinfo color="primary" %}}

**Read the [API documentation &raquo;](https://pkg.go.dev/github.com/AdRoll/baker/input#Kinesis)**
{{% /pageinfo %}}

## Input *Kinesis*

### Overview
This input fetches log lines from Kinesis.  
 It listens on a specified stream, and
processes all the shards in that stream.  
 It never exits.  



### Configuration

Keys available in the `[input.config]` section:

|Name|Type|Default|Required|Description|
|----|:--:|:-----:|:------:|-----------|
| AwsRegion| string| "us-west-2"| false| AWS region to connect to|
| Stream| string| ""| true| Stream name on Kinesis|
| IdleTime| duration| 100ms| false| Time between polls of each shard|

