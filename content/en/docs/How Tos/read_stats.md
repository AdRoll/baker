---
title: "Read statistics"
date: 2020-11-16
weight: 775
description: >
  How to read Baker stats
---

While running, Baker dumps statistics on stdout every second. This is an example line:

```log
Stats: 1s[w:29425 r:29638] total[w:411300 r:454498 u:1831] speed[w:27420 r:30299] errors[p:0 i:0 f:0 o:0 u:0]
```

The first bracket shows the current read and write speed, i.e. the records that entered the
pipeline (reads) and the records that successfully exited from it (writes).

The second bracket is the total since the process was launched (the `u:` key is the number of
files successfully uploaded).

The third bracket shows the average read/write speed (records per second).

The fourth bracket shows the number of records that were discarded at some point because of errors:

* `p:` records discarded for a parsing error
* `i:` input records discarded. Most of the time, this refers to validation issues.
* `f:` records discarded by filters.
* `o:` is the number of records that were discarded because of an error in the output component *
* `u:` is the number files whose upload has failed

\* Notice that output components should be resilient to transient network failures,
and they abort the process in case of permanent configuration errors, so the number
here reflects records that could not be permanently written because eg. validation
issues. Eg. think of an output that expects a column to be in a specific format, and
rejects records where that field is not in the expected format. A real-world example
is empty columns that are not accepted by DynamoDB.
