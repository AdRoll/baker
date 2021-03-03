---
title: "SQS"
weight: 5
date: 2021-03-03
---
{{% pageinfo color="primary" %}}

**Read the [API documentation &raquo;](https://pkg.go.dev/github.com/AdRoll/baker/input#SQS)**
{{% /pageinfo %}}

## Input *SQS*

### Overview
This input listens on multiple SQS queues for new incoming log files
on S3; it is meant to be used with SQS queues popoulated by SNS.
It never exits.


### Configuration

Keys available in the `[input.config]` section:

|Name|Type|Default|Required|Description|
|----|:--:|:-----:|:------:|-----------|
| AwsRegion| string| "us-west-2"| false| AWS region to connect to|
| Bucket| string| ""| false| S3 Bucket to use for processing|
| QueuePrefixes| array of strings| []| true| Prefixes of the names of the SQS queues to monitor|
| MessageFormat| string| "sns"| false| The format of the SQS messages.
'plain' the SQS messages received have the S3 file path as a plain string.
'sns' the SQS messages were produced by a SNS notification.|
| FilePathFilter| string| ""| false| If provided, will only use S3 files with the given path.|

