---
title: "Create a custom input component"
date: 2020-11-02
weight: 650
---
The job of a Baker input is to fetch blob of data containing one or multiple serialized records
and send them to Baker.

The input isn't in charge of splitting/parsing the input data into Records (that is done by Baker),
but only retrieving them as fast as possible in raw format adding, if any, metadata to them and
then sending those values to Baker through a
[`*Data`](https://pkg.go.dev/github.com/AdRoll/baker#Data) channel. The channel size is
customizable in the topology TOML with `[input] chansize=<value>` (default to 1024).

To create an input and make it available to Baker, one must:

* Implement the [Input](https://pkg.go.dev/github.com/AdRoll/baker#Input) interface
* Fill an [`InputDesc`](https://pkg.go.dev/github.com/AdRoll/baker#InputDesc) structure and register it
within Baker via [`Components`](https://pkg.go.dev/github.com/AdRoll/baker#Components).

## Daemon vs Batch

The input component determines the Baker behavior between a batch processor or a long-living daemon.

If the input exits when its data processing has completed, then Baker waits for the topology to end
and then exits.

If the input never exits, then Baker acts as a daemon.

## Data

The [`Data`](https://pkg.go.dev/github.com/AdRoll/baker#Data) object that the input must fill in
with read data has two fields: `Bytes`, that must contain the raw read bytes (possibly containing
more records separated by `\n`), and `Meta`.

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
	Help:   "High-level description of MyInput",
}
```

This object has a `Name`, that is used in the Baker configuration file to identify the input,
a costructor-like function (`New`), a config object (where the parsed input configuration from the
TOML file is stored) and a help text that must help the users to use the component and its
configuration parameters.

### Input constructor-like function

The `New` key in the `InputDesc` object represents the constructor-like function.

The function receives a [InputParams](https://pkg.go.dev/github.com/AdRoll/baker#InputParams)
object and returns an instance of [Input](https://pkg.go.dev/github.com/AdRoll/baker#Input).

The function should verify the configuration params into `InputParams.DecodedConfig` and initialize
the component.

### The input configuration and help

The input configuration object (`MyInputConfig` in the previous example) must export all
configuration parameters that the user can set in the TOML topology file.

Each field in the struct must include a `help` string tag (mandatory) and a `required` boolean tag
(default to `false`).

All these parameters appear in the generated help. `help` should describe the parameter role and/or
its possible values, `required` informs Baker it should refuse configurations in which that field
is not defined.

## Write tests

To test an input component we suggest two main paths:

* test the component in isolation, calling the `Run` function
* test the input at high-level, running a complete Baker topology

Regardless of the chosen path, two additional unit tests are always suggested:

* test the `New()` (constructor-like) function, to check that the function is able to correctly
instantiate the component with valid configurations and intercept wrong ones
* create small and isolated functions where possible and unit-test them

### Test calling Run()

In case we want to test the component calling the `Run` function, this is an example of test where,
after some initialization, the `input.Run` function is called and the produced `Data` is checked
in a goroutine:

```go
func TestMyInput(t *testing.T) {
    ch := make(chan *baker.Data)
    defer close(ch)

    go func() {
        for data := range ch {
            // test `data`, that comes from the component,
            // like checking its content, parse the records, metadata, etc
            if something_is_wrong(data) {
                t.Fatalf("error!")
            }
        }
    }()

    // configure the input
    cfg := ...

    input, err := NewMyInput(cfg) // use the contructor-like New function
    // check err

    // if the input requires other things, initialize/create them

    // run the input
    if err := input.Run(ch); err != nil {
        t.Fatal(err)
    }
}
```

The `List` input [has an example](https://github.com/AdRoll/baker/blob/main/input/list_test.go)
of this testing strategy.

### Test the component running a topology

If we want to test the component creating and running a topology, we need to create one starting
from the TOML configuration and then calling `NewConfigFromToml`, `NewTopologyFromConfig` and `Run`.

The `Base`, `Recorder` and `RawRecorder` outputs included in the
[`outputtest` package](https://github.com/AdRoll/baker/tree/main/output/outputtest) can be
helpful here to obtain the output and check it:

```go
func TestMyInput(t *testing.T) {
    toml := `
    [input]
    name = "MyInput"

    [output]
    name="RawRecorder"
    procs=1
    `
    // Add the input to be tested and a testing output
    c := baker.Components{
        Inputs:  []baker.InputDesc{MyInputDesc},
        Outputs: []baker.OutputDesc{outputtest.RawRecorderDesc},
    }

    // Create and start the topology
    cfg, err := baker.NewConfigFromToml(strings.NewReader(toml), c)
    if err != nil {
        t.Error(err)
    }
    topology, err := baker.NewTopologyFromConfig(cfg)
    if err != nil {
        t.Error(err)
    }
    topology.Start()
    
    // In this goroutine we should provide some inputs to the component
    // The format and how to send them to the component, depends on
    // the component itself
    go func() {
        defer topology.Stop()
        sendDataToMyInput() // fake function, you need to implement your logic here
    }

    topology.Wait() // wait for Baker to quit after `topology.Stop()`
    if err := topology.Error(); err != nil {
        t.Fatalf("topology error: %v", err)
    }

    // retrieve the output and test the records
    out := topology.Output[0].(*outputtest.Recorder)
    if len(out.Records) != want {
        t.Errorf("want %d log lines, got %d", want, len(out.Records))
    }

    // more testing on out.Records...
}
```

The `TCP` input [includes an example](https://github.com/AdRoll/baker/blob/main/input/tcp_test.go)
of this testing strategy.
