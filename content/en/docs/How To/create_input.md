---
title: "Create a custom input component"
date: 2020-11-02
weight: 650
---
The job of a Baker input is to fetch blob of data containing one or multiple serialized records and send them to Baker.

The input isn't in charge of splitting/parsing the input data into Records (that is done by Baker),
but only retrieving them as fast as possible in raw format adding, if any, metadata to them.

The input shares a [`*Data`](https://pkg.go.dev/github.com/AdRoll/baker#Data) channel with the
filter chain. The channel size is customizable with `[input] chansize=<value>` (default to 1024).

To create an input and make it available to Baker, one must:

* Implement the [Input](https://pkg.go.dev/github.com/AdRoll/baker#Input) interface
* Fill an [`InputDesc`](https://pkg.go.dev/github.com/AdRoll/baker#InputDesc) structure and register it
within Baker via [`Components`](https://pkg.go.dev/github.com/AdRoll/baker#Components).
the available inputs in [`Components`](https://pkg.go.dev/github.com/AdRoll/baker#Components)

## Daemon vs Batch

The input component determines the Baker behavior between a batch processor or a long-living daemon.

If the input exits when its data processing has completed, then Baker waits for the topology to end
and then exits.

If the input never exits, then Baker acts as a daemon.

## Data

The [`Data`](https://pkg.go.dev/github.com/AdRoll/baker#Data) object that the input must fill in
with read data has two fields: `Bytes` that must contain the raw read bytes (possible contains
more records) and `Meta`.

[`Metadata`](https://pkg.go.dev/github.com/AdRoll/baker#Metadata) can contain additional 
information Baker will associate with each of the serialized Record contained in `Data`.  
Typical information could be the time of retrieval, the filename (in case `Records` come from a file), etc.

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

`FreeMem(data *Data)` is called by Baker when `data` is no longer needed. This is an occasion
for the input to recycle memory, for example if the input uses a `sync.Pool` to create new 
instances of `baker.Data`. 

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
