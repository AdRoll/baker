---
title: "Performance"
weight: 150
date: 2020-11-20
description: >
  Baker performance test cases
---

Baker has been designed with high performance in mind. Baker core, the part of the code base
which distributes records among components and ties them together, is very high-quality Go code.
Records are never copied, and a particular attention has been given to reduce the number of
memory allocations as much as possible, so as to keep the garbage collector cost to a minimum.

Baker is also battle-tested, since 2016 NextRoll has been running hundreds if not thousands
of Baker pipelines, processing petabytes, daily.

We report in this page some practical examples of Baker performances we could measure in the
NextRoll production environment.

Within NextRoll, Baker is often executed on AWS EC2 instances, and thus you find in this page
many references to
[EC2 instance types](https://aws.amazon.com/ec2/instance-types/) (`c5.2xlarge`, `c5.2xlarge`, etc.).

### Read from S3 and write to local disk

On an AWS EC2 instance of size `c5.2xlarge`, Baker can read zstandard records from S3, uncompress
them and apply a basic filtering logic, compressing them back on local files using ~90% of capacity
of each vCPU (8 in total) and
~3.5GB of RAM.  

It reads and writes a total of 94 million records in less than 9 minutes, that's 178k records per
second.

On a `c5.2xlarge` instance (48 vCPUs) the same test takes 2 minutes, so that's a speed of 775k
records per second.

For this test we use 711 zstd compressed files for a total of 17 GB of compressed size and 374 GB
of uncompressed size. The average size of each record is 4.5 KB.

### Read from S3 and write to DynamoDB (in the same region)

On a `c5.4xlarge` instance, Baker reads zstd compressed files from S3 writing to DynamoDB
(configured with 20k write capacity units) at an average speed of 60k records/s (the average size of
each record is 4.3 KB) using less than 1 GB of memory and 300% of the total CPU capacity (less than
20% for each core).

The bottleneck here is the DynamoDB write capacity, so Baker could handle the additional load caused
by a possible increase in the write capacity.

### Read from Kinesis and write to DynamoDB (in the same region)

On a `c5.4xlarge` instance, we performed a test reading from a Kinesis stream with 130 shards and
writing to a DynamoDB table with 20k write capacity units. Baker is able to read and write more
than 10k records per second (the average size of each record is 4.5 KB) using less than 1 GB of RAM
and around 400% of the total CPU capacity (each core being used at less than 25%).
