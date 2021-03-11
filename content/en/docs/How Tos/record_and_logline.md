---
title: "Record and LogLine"
date: 2020-10-29
weight: 200
description: >
  Baker operates on objects called "records", an abstraction over integer-indexable structured data.
---

Baker processes objects called [**records**](http://localhost:1313/docs/core-concepts/#record-and-logline).
A `Record`, in Baker, is an interface that provides an abstraction over a record of 
structured data, where fields are indexed and accessed via integers.

At the moment, Baker provides a single implementation of the Record interface,
called [`LogLine`](https://pkg.go.dev/github.com/AdRoll/baker#LogLine).
If `LogLine` doesn't fit your needs, you can implement the 
[`Record` interface](https://pkg.go.dev/github.com/AdRoll/baker#Record) or 
modify [`LogLine`](#custom-logline). See the [custom Record how-to](/docs/how-tos/custom_record/)
for more details about implementing the `Record` interface from scratch.

## LogLine

A [`LogLine`](https://pkg.go.dev/github.com/AdRoll/baker#LogLine) is an implementation
of the [`Record` interface](https://pkg.go.dev/github.com/AdRoll/baker#Record)
which is highly optimized for fast parsing and serializing of CSV records.
It supports any single-byte field separator and doesn't handle quotes (neither single nor double).  

The maximum number of fields is hard-coded by the `LogLineNumFields` constant which is 3000.
100 extra fields can be stored at runtime in a `LogLine` (also hardcoded with `NumFieldsBaker`),
these extra fields are a fast way to exchange data between filters and/or outputs but they are
neither handled during Parsing (i.e `LogLine.Parse`) nor serialization (`LogLine.ToText`).

### Custom LogLine

If the hardcoded values for
[`LogLineNumFields` and `NumFieldsBaker`](https://pkg.go.dev/github.com/AdRoll/baker#pkg-constants)
do not suit your needs, you can copy [`logline.go`](https://github.com/AdRoll/baker/blob/main/logline.go)
in your project and modify the constants declared at the top of the file.

Your specialized `LogLine` will still implement `baker.Record` and thus can be used in lieu
of `baker.LogLine`.
The `CreateRecord` function set into
[`baker.Components`](https://pkg.go.dev/github.com/AdRoll/baker#Components) must return an
instance of your custom `LogLine` instead of the default one.
