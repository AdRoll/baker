---
title: "Create a custom input component"
date: 2020-11-02
weight: 650
---
The input component goal is to retrieve Records in form of bytes and send them to the filterchain.

The input isn't in charge of splitting/parsing the input data into Records (that is done by Baker),
but only retrieving them as fast as possible in raw format adding, if any, metadata to them.

The input shares a [`*Data`](https://pkg.go.dev/github.com/AdRoll/baker#Data) channel with the
filter chain. The channel size is customizable with `[input] chansize=<value>` (default to 1024).

To create an input and make it available to Baker, one must:

* Implement the [Input](https://pkg.go.dev/github.com/AdRoll/baker#Input) interface
* Add an [`InputDesc`](https://pkg.go.dev/github.com/AdRoll/baker#InputDesc) for the input to
the available inputs in [`Components`](https://pkg.go.dev/github.com/AdRoll/baker#Components)

## Daemon vs Batch

The input component determines the Baker behavior between a batch processor or a long-living daemon.

If the input exits when its data processing has completed, then Baker waits the topology to end
and then exits.

If the input never exits, then Baker acts as a daemon.

## Data

The [`Data`](https://pkg.go.dev/github.com/AdRoll/baker#Data) object that the input must fill in
with read data has two fields: `Bytes` that must contain the raw read bytes (possible contains
more records) and `Meta`.

[`Metadata`](https://pkg.go.dev/github.com/AdRoll/baker#Metadata) can contain any information
that the input wants to share with the other components of the pipeline about the chunk of bytes
produced.  
Typical information could be the timestamp of the read, details about the source, etc. 

Those metadata will be associated to each record extracted from `Bytes` and will be available
to the whole pipeline.

## The Input interface

```go
type Input interface {
	Run(output chan<- *Data) error
	Stop()
	Stats() InputStats
	FreeMem(data *Data)
}
```

The [Input interface](https://pkg.go.dev/github.com/AdRoll/baker#Input) must be implemented when
creating a new input component.

The `Run` function implements the component logic and receives a channel where it sends the
[raw data](https://pkg.go.dev/github.com/AdRoll/baker#Data) it processes.

`FreeMem` is the function called by Baker when a chunk of data has been completely processed and
its memory can be used again. It's up to the input to choose whether to recycle it or not.  
An often used pattern is to manage the objects with [`sync.Pool`](https://golang.org/pkg/sync/#Pool),
see the [TCP](https://github.com/AdRoll/baker/blob/main/input/tcp.go),
[List](https://github.com/AdRoll/baker/blob/main/input/list.go) or
[SQS](https://github.com/AdRoll/baker/blob/main/input/sqs.go) inputs for inspiration.

## InputDesc

```go
var MyInputDesc = baker.InputDesc{
	Name:   "MyInput",
	New:    NewMyInput,
	Config: &MyInputConfig{},
	Help:   "Help message for the input",
}
```

This object has a `Name`, that is used in the Baker configuration file to identify the input,
a costructor function (`New`), a config object (used to parse the input configuration in the
TOML file) and a help text that must help the users to use the component and its configuration
parameters.

### Input constructor

The `New` key in the `InputDesc` object represents the constructor function.

The function receives a [InputParams](https://pkg.go.dev/github.com/AdRoll/baker#InputParams)
object and returns an instance of [Input](https://pkg.go.dev/github.com/AdRoll/baker#Input).

The function should verify the configuration params into `InputParams.DecodedConfig` and initialize
the component. Additional operations can be performed in the `Run` function when called.

### The input configuration and help

The input configuration object (`MyInputConfig` in the previous example) must export all
configuration parameters that the user can set in the TOML topology file.

Each key in the struct must include a `help` string tag and a `required` boolean tag, both are
mandatory.

The former helps the user to understand the possible values of the field, the latter tells Baker
whether to refuse a missing configuration param.
