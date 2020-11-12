---
title: "KCL"
weight: 2
date: 2020-11-12
---
## Input *KCL*

### Overview
This input fetches records from Kinesis with KCL.
 It consumes a specified stream, and
processes all shards in that stream.
 It never exits.

Multiple baker instances can consume the same stream, in that case the KCL will take care of
balancing the shards between workers.
 Careful (shard stealing is not implemented yet).

Resharding on the producer side is automatically handled by the KCL that will distribute
the shards among KCL workers.


### Configuration

Keys available in the `[input.config]` section:

|Name|Type|Default|Required|Description|
|:--:|:--:|:-----:|:------:|:---------:|
| AwsRegion| string| "us-west-2"| false| AWS region to connect to|
| Stream| string| ""| true| Name of Kinesis stream|
| AppName| string| ""| true| Used by KCL to allow multiple app to consume the same stream.|
| MaxShards| int| 32767| false| Max shards this Worker can handle at a time|
| ShardSync| duration| 60s| false| Time between tasks to sync leases and Kinesis shards|
| InitialPosition| string| "LATEST"| false| Position in the stream where a new application should start from. Values: LATEST or TRIM_HORIZON|

