---
title: "Create a custom filter component"
date: 2020-11-02
weight: 700
---

Creating a custom filter is probably the most common action a Baker user will perform.

In fact, filters are the components that apply the business logic to a Baker pipeline,
**creating or discarding records or modifying fields.**

A working example of a custom filter can be found in the
[filtering example](https://github.com/AdRoll/baker/tree/main/examples/filtering)

To create a filter and make it available to Baker, one must:

* Implement the Filter interface
* Add a [`FilterDesc`](https://pkg.go.dev/github.com/AdRoll/baker#FilterDesc) for the filter to
the available filters in [`Components`](https://pkg.go.dev/github.com/AdRoll/baker#Components)


## The Filter interface

The [Filter interface](https://pkg.go.dev/github.com/AdRoll/baker#Filter) determines the functions
that a filter must implement. The interface is quite simple and contains only two functions:
`Process(l Record, next func(Record))` and `Stats() FilterStats`:

* `Process` is the function the actually filters the records
* `Stats` return statistics ([FilterStats](https://pkg.go.dev/github.com/AdRoll/baker#FilterStats)) about the filtering process

A very simple example of filter doing nothing is:

```go
type MyFilter struct{
    numProcessedLines int64
}

func (f *MyFilter) Process(r Record, next func(Record)) {
    atomic.AddInt64(&f.numProcessedLines, 1)
    next(r)
}

func (f *MyFilter) Stats() FilterStats { 
    return baker.FilterStats{
		NumProcessedLines: atomic.LoadInt64(&f.numProcessedLines),
    }
}
```

## FilterDesc

To be included in the Baker filters, a filter must be described by a
[`FilterDesc` object](https://pkg.go.dev/github.com/AdRoll/baker#FilterDesc):

```go
var MyFilterDesc = baker.FilterDesc{
	Name:   "MyFilter",
	New:    NewMyFilter,
	Config: &MyFilterConfig{},
	Help:   "This filter does nothing, but in a great way!",
}
```

This object has a `Name`, that is used in the Baker configuration file to identify the filter,
a costructor function (`New`), a config object (used to parse the filter configuration in the
TOML file) and a help text.

In this case the filter can be used with this configuration in the
[TOML file](/docs/how-to/pipeline_configuration/):

```toml
[[filter]]
name = "MyFilter"
```

### Filter constructor

Each filter must have a constructor function that receives a
[FilterParams](https://pkg.go.dev/github.com/AdRoll/baker#FilterParams) and returns the 
[Filter interface](https://pkg.go.dev/github.com/AdRoll/baker#Filter) implemented by the filter:

```go
func MyFilter(cfg baker.FilterParams) (baker.Filter, error) {
	return &MyFilter{}, nil
}
```

The [filtering example](https://github.com/AdRoll/baker/blob/main/examples/filtering/filter.go)
shows a more complex constructor that also uses the `FilterParams` argument.

### The filter configuration and help

A filter requiring some configurations also has a config object, including as many keys as it
needs and tagging each one with an `help` tag, a string that contains what a user needs to know
which values set for it:

```go
type ClauseFilterConfig struct {
	Clause string `help:"Boolean formula describing which events to let through. If empty, let everything through."`
}
```

## Modify record fields

A filter can change the value of the record fields before calling `next()`:

```go
func (f *MyFilter) Process(r Record, next func(Record)) {
    var src FieldIndex = 10
    var dst FieldIndex = 10
    v := r.Get(src)
    //.. modify v as required
    r.Set(dst, v)
    next(r)
}
```

## Processing records

Filters do their work in the `Process(r Record, next func(Record)` method, where `r` is the
Record to process and `next` is a closure assigned to the next element in thefilter chain.

Filters call `next(r)` once they're done with the record and desire to forward it, or simply
do not call `next()` if they want to discard the record.

When a filter discards a record it should also report it in the stats:

```go
type MyFilter struct{
    numProcessedLines int64
    numFilteredLines  int64
}

func (f *MyFilter) Process(r Record, next func(Record)) {
    atomic.AddInt64(&f.numProcessedLines, 1)

    // shouldBeDiscarded is part of the filter logic
    if shouldBeDiscarded(r) {
        atomic.AddInt64(&f.numFilteredLines, 1)
        // return here so next() isn't called
        return
    }
    // forward the record to the next element of the filter chain
    next(r)
}

func (f *MyFilter) Stats() FilterStats { 
    return baker.FilterStats{
        NumProcessedLines: atomic.LoadInt64(&f.numProcessedLines),
        NumFilteredLines:  atomic.LoadInt64(&f.numFilteredLines),
    }
}
```

## Create records

A filter can decide to call `next()` multiple times to send new or duplicated records to the
next element of the filter chain.

Note that the new or copied records don't start the filter chain from the first filter in the list
but only the remaining filters are applied to the records.

{{% alert color="warning" %}}
Remember not to pass the same record to multiple `next()` functions or later changes to one of
the records could also impact the others.  
Always use `Copy()` or `CreateRecord()` before calling `next()` more than once.
{{% /alert %}}

### Copy()

Filters can duplicate incoming records (with `record.Copy()`), and thus have more records
come out than records that came in.

```go
func (f *MyFilter) Process(r Record, next func(Record)) {
    // Call next the 1st time
    next(r)

    // WRONG, it is the same record as above
    next(r)

    // CORRECT, this is a copy of the record
    next(r.Copy())
}
```

### CreateRecord()

A new, empty, record is created calling the `CreateRecord` function.
The `CreateRecord` function is available as part of the
[FilterParams](https://pkg.go.dev/github.com/AdRoll/baker#FilterParams) argument of the
[filter constructor](#filter-constructor). If you plan to use it in the `Process` function
then store it to the filter object in the constructor as shown in this example:

```go
type MyFilter struct{
    cfg baker.FilterParams
}

func NewMyFilter(cfg baker.FilterParams) (baker.Filter, error) {
	return &MyFilter{
        cfg: cfg, // you can also store only CreateRecord
    }, nil
}

func (f *MyFilter) Process(r Record, next func(Record)) {
    newRecord := f.cfg.CreateRecord()
    //... do something with the record
    next(newRecord)
}
```
