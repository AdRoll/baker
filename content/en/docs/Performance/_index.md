---
title: "Performance"
weight: 150
date: 2020-11-20
description: >
  Baker performance test cases
---

### Read from S3 and write to local disk

On a `c5.2xlarge` instance, Baker managed to read zstandard records from S3, uncompress them and
apply a basic filtering logic, compressing them back on local files with zstandard at compression
level 3 and long range mode at 27 using ~90% of capacity of each vCPU (8 in total) and ~3.5GB of RAM.  
It read and wrote a total of 94 million records in 8'51" (~178k r/w records per second).  
On a `c5.12xlarge` instance (48vCPUs) the same test took 2'2" (~775k r/w records per second).

For this test we used 711 zstd compressed files for a total of 17 GB of compressed size and 374 GB
of uncompressed size. The average size of each record was ~4.5 KB.

### Read from S3 and write to DynamoDB (in the same region)

On a `c5.4xlarge` instance, Baker read zstd compressed files from S3 writing to DynamoDB (configured
with 20k write capacity units) at an average speed of 60k records/s (the average size of each record
is 4.3 KB) using less than 1 GB of memory and ~300% of the total CPU capacity (less than 20% for
each core). The bottleneck here was the DynamoDB write capacity, so Baker can easily cope with an
increased load just increasing the write capacity units in DynamoDB (up to 400k).

### Read from Kinesis and write to DynamoDB (in the same region)

On a `c5.4xlarge` instance, we performed a test reading from a Kinesis stream with 130 shards and
writing to a DynamoDB table with 20k write capacity units. Baker was able to read and write more
than 10k records per second (the avg size of each record was 4.5 KB) using less than 1 GB of RAM and
~400% of the total CPU capacity (less than 25% for each core).
