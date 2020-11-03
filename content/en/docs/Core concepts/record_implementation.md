---
title: "Record implementation"
date: 2020-10-29
weight: 2
description: >
  Record is the interface representing the entries processed by Baker. This page explains how to
  implement a custom record.
---

Baker processes objects in form of "records". A Record, in Baker, is an interface that
provides an abstraction over a record of flattened data, where columns of fields are
indexed through integers.

Baker currently provides a single implementation of Record, called
[`LogLine`](https://pkg.go.dev/github.com/AdRoll/baker#LogLine).  
If `LogLine` doesn't fit your needs, you can easily implement the Record interface with your
logic.

## Record interface

This is the [Record interface](https://pkg.go.dev/github.com/AdRoll/baker#Record) as
declared in `record.go`. Each method is explained below:

```go
type FieldIndex int
type Metadata map[string]interface{}

type Record interface {
	Parse([]byte, Metadata) error
	ToText(buf []byte) []byte
	Copy() Record
	Clear()
	Get(FieldIndex) []byte
	Set(FieldIndex, []byte)
	Meta(key string) (v interface{}, ok bool)
	Cache() *Cache
}
```

### Parse([]byte, Metadata) error

The `Parse` creates a Record instance by deserializing a slice of byte.

It also receives a, possibly nil, `Metadata` map that the input can fill in (like record retrieval
timestamp or any other useful info). The function must be able to accept a `nil` Metadata value.

### ToText(buf []byte) []byte

`ToText` serializes the Record into a slice of bytes.

In case the passed `buf` is not `nil` (and if big enough), it is used to serialize the record.

### Copy() Record

`Copy` creates and returns a deep-copy of the record.
	
There is a "simple" way to create a copy of a record:

```go
var dst Record
dst.Parse(src.ToText(), nil)
```

but the `Copy` function should provide a more efficient way, optimized for the
custom record implementation.

### Clear()

`Clear` clears the record internal state, making it empty and re-usable as an empty record.

### Get(FieldIndex) []byte

`Get` returns the value of a field at the given index.

### Set(FieldIndex, []byte)

Set the value of a field at the give index.

### Meta(key string) (v interface{}, ok bool)

`Meta` returns the value of the attached metadata for the given key, if any.

The simplest implementation could be:

```go
type MyRecord struct {
    meta baker.Metadata
}

func (r *MyRecord) Meta(key string) (interface{}, bool) {
    return l.meta.get(key)
}
```

### Cache() *Cache

`Cache` holds a cache which is local to the record.  
It may be used to speed up parsing of specific fields by caching the result.  
When accessing a field and parsing its value, we want to try caching as much as
possible the parsing we do, to avoid redoing it later when the same record
is processed by different code.

Since cached values are interfaces, it's up to who fetches a value to know the underlying
type of the cached value and perform a type assertion.

```go
var ll Record
val, ok := ll.Cache.Get("mykey")
if !ok {
    // long computation/parsing...
    val = "14/07/1789"
    ll.Cache.Set("mykey", val)
}

// do something with the result
result := val.(string)
```

## LogLine

A `LogLine` is a highly optimized CSV. It supports any single-byte field separator and doesn't
handle quotes (neither single nor double).  
The maximum number of fields is hard-coded by the `LogLineNumFields` constant which is 3000.  
100 extra fields can be stored at runtime in a `LogLine` (also hardcoded with `NumFieldsBaker`),
these extra fields are a fast way to exchange data between filters and/or outputs but they are
neither handled during Parsing (i.e `LogLine.Parse`) nor serialization (`LogLine.ToText`).

### Custom LogLine
If the hardcoded values for `LogLineNumFields` and `NumFieldsBaker` do not suit your needs,
it's advised that you copy `LogLine.go` in your project and modify the constants declared at
the top of the file. Your specialized `LogLine` will still implement `baker.Record` and thus
can be used in lieu of `baker.LogLine`. To do so, you need to provide a `CreateRecord`
function to `baker.Components` when calling `baker.NewConfigFromToml`.

For example (`my.LogLine` is your custom implementation):

```go
comp := baker.Components{}

comp.CreateRecord = func() baker.Record {
  return &my.LogLine{ FieldSeparator: ',' }
}
```

## How to use a custom version of the Record

Once a customized version of a Record has been implemented, you want to use it in your code.
In order to do so, some functions may be implemented while instantiating `baker.Components`:

```go
type Components struct {
	Validate      ValidationFunc
    CreateRecord  func() Record
    FieldByName func(string) (FieldIndex, bool)
    FieldName   func(FieldIndex) string
    //... other functions
}
```

### Validate

`Validate` is the function used to validate a record. It is called for each processed record
unless not set or when the `[general] dont_validate_fields = true` configuration is set in
the TOML file.

Regardless of the TOML configuration, the function is passed to all components that can use
it at their will.

### CreateRecord

`CreateRecord` is the function that creates a new record. If not set, a default function is
used that creates a `LogLine` with `,` as field separator.

The function is used internally by Baker to create new records every time a new one comes from
the input.

The function is also passed to components that can use it to create new records while processing.

### FieldByName

`FieldByName` gets a field index by its name. The function is mainly used by the components
(that receive it during setup) to retrieve the index of a field they need for filtering or
processing, but it is also used internally by Baker when sending fields to the output
(when at least one field is selected in the output TOML configuration).

### FieldIndex

`FieldName` gets a field name by its index. The function is passed to components that can use
it for their internal logic.

## RecordConformanceTest

The `test_helper.go` provides a `RecordConformanceTest` test helper whose goal is to give the
user a structured test for new implementations of the Record.

The helper receives the implementation of `CreateRecord` and creates new records testing
them against a set of requirements.

{{% alert title="Warning" color="warning" %}}
The conformance test provides a way to verify that a record implementation respects the
invariant that Baker requires for a Record implementation and thus it should always
be executed against all custom implementations of the Record.
{{% /alert %}}
