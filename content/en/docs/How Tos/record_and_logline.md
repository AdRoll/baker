---
title: "Record and LogLine"
date: 2020-10-29
weight: 200
description: >
  Baker deals with "record" objects, an abstraction of flattened data with indexed fields.
---

Baker processes objects in form of "records". A Record, in Baker, is an interface that
provides an abstraction over a record of flattened data, where columns of fields are
indexed through integers.

Baker currently provides a single implementation of Record, called `LogLine` (
[API reference](https://pkg.go.dev/github.com/AdRoll/baker#LogLine)).

If `LogLine` doesn't fit your needs, you can easily implement the Record interface with your
[custom logic](#custom-logline).

## LogLine

A [`LogLine`](https://pkg.go.dev/github.com/AdRoll/baker#LogLine) is a Record implementation 
which is highly optimized for fast parsing and serializing of CSV records.  

It supports any single-byte field separator and doesn't handle quotes (neither single nor double).  

The maximum number of fields is hard-coded by the `LogLineNumFields` constant which is 3000.  

100 extra fields can be stored at runtime in a `LogLine` (also hardcoded with `NumFieldsBaker`),
these extra fields are a fast way to exchange data between filters and/or outputs but they are
neither handled during Parsing (i.e `LogLine.Parse`) nor serialization (`LogLine.ToText`).

### Custom LogLine

If the hardcoded values for
[`LogLineNumFields` and `NumFieldsBaker`](https://pkg.go.dev/github.com/AdRoll/baker#pkg-constants)
do not suit your needs, it's advised that you copy [`logline.go`](https://github.com/AdRoll/baker/blob/main/logline.go)
in your project and modify the constants declared at the top of the file.

Your specialized `LogLine` will still implement `baker.Record` and thus can be used in lieu
of `baker.LogLine`.

The `CreateRecord` function set into
[`baker.Components`](https://pkg.go.dev/github.com/AdRoll/baker#Components) must return an
instance of your custom LogLine instead of the default one.
