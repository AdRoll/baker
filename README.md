# Baker

Baker is a high performance, composable and extendable data-processing pipeline
for the big data era. It shines at converting, processing, extracting or storing
records (structured data), applying whatever transformation between input and
output through easy-to-write filters.

Baker is fully parallel and maximizes usage of both CPU-bound and I/O bound pipelines.
On a AWS c3.4xlarge instance (16vCPU amd64 / 30GB RAM), it can run a simple pipeline
(with little processing of records) achieving 30k writes per seconds to DynamoDB in 4
AWS regions, using ~1GB of RAM in total, and 500% of CPU (so with room for scaling
even further if required).

## Pipelines

A pipeline is the configured set of operations that Baker performs during its execution.

It is defined by:

* One input component, defining where to fetch records from
* Zero or more filters, which are functions that can process records (reading/writing
  fields, clearing them or even splitting them into multiple records)
* One output component, defining where to send the filtered records to (and which
  columns)
* One optional upload component, defining where to send files produced by the output
  component (if any)

Notice that there are two main usage scenarios for Baker:

 1. Baker as a batch processor. In this case, Baker will go through all the records
    that are fed by the input component, process them as quickly as possible, and exit.
 2. Baker as a daemon. In this case, Baker will never exit; it will keep waiting for
    incoming records from the input component (e.g.: Kinesis), process them and send
    them to the output.

Selecting between scenario 1 or scenario 2 is just a matter of configuring the pipeline;
in fact, it is the input component that drives the scenario. If the input component
exits at some point, Baker will flush the pipeline and exit as well; if instead the
input component is endless, Baker will never exit and thus behave like a daemon.

## Usage

Baker uses a `baker.Config` struct to know what to do. The configuration can be created
parsing a toml file with `baker.NewConfigFromToml()`. This function requires a
`baker.Components` object including all available/required components.  
This is an example of this struct:

```go
baker.Components{
    Inputs:        input.AllInputs(),
    Filters:       MyCustomFilters(),
    // merge available outputs with user custom outputs
    Outputs:       append(output.AllOutputs(), MyCustomOutputs()...),
    Uploads:       MyCustomUploads(),
    // optional: custom extra config
    User:          MyCustomConfigs(),
    // optional: used if sharding is enabled
    ShardingFuncs: MyRecordShardingFuncs,
    // optional: used if records must be validated. Not used if [general.dont_validate_fields] is used in TOML
    Validate:      MyRecordValidationFunc,
    // optional: functions to get fields indexes by name and vice-versa
    FieldByName:   MyFieldByNameFunc,
    FieldName:     MyFieldNameFunc,
}
```

Components (inputs, filters, outputs and uploads) are either generic ones provided with Baker or user-defined components.
provided by Baker can be used. They can also be merged in a single slice to make
all of them available.

## How to build a Baker executable

The `examples/` folder contains several `main()` examples:

* [basic](./examples/basic/): a simple example with minimal support
* [filtering](./examples/filtering/): shows how to code your own filter
* [sharding](./examples/sharding/): shows how to use an output that supports sharding
  (see below for details about sharding)
* [help](./examples/help/):  shows components' help messages
* [advanced](./examples/advanced/): an advanced example with most of the features supported by Baker

## TOML Configuration files

This is a minimalist Baker pipeline TOML configuration that reads a record from the disk,
updates its timestamp field with a "Timestamp" filter and pushes it to DynamoDB:

```toml
[input]
name="List"

    [input.config]
    files=["records.csv.gz"]

[[filter]]
name="Timestamp"

[output]
name="DynamoDB"
fields=["source","timestamp","user"]


    [output.config]
    regions=["us-west-2","us-east-1"]
    table="TestTableName"
    columns=["s:Source", "n:Timestamp", "s:User"]
```

`[input]` selects the input component, or where to read the records from. In this case,
the `List` component is selected, which is a component that fetches logs from a list of
local or remote paths/URLs. `[input.config]` is where component-specific configurations
can be specified, and in this case we simply provide the `files` option to `List`. Notice
that `List` would accept `http://` or even `s3://` URLs there in addition to local paths,
and some more (run `./baker-bin -help List` in the [help example](./examples/help/) for
more details).

`[[filter]]` selects which filters to run. In TOML syntax, the double brackets indicate
an array of sections, so it basically means that you can define many different `[[filter]]`
sections, one for each filter that you wish to run. In this example, only the `Timestamp`
filter is selected. This configuration file doesn't specifiy any option for this filter;
if needed, those options would go to a `[filter.config]` subsection.

`[output]` selects the output component, the output is where the records that made up until the end of the filter chain without being discarded end up.
In this case, the `DynamoDB` component is selected, and its configuration is specified
in `[output.config]`.

The `fields` option in the `[output]` section selects which fields of the record will be
send to the output. In fact, most pipelines don't want to send the full records to the
output, but they will select a few important columns out of the many available columns.
Notice that this is just a selection: it is up to the output component to decide how to
physically serialize those columns. For instance, the `DynamoDB` component requires the
user to specify an option called `columns` that specifies the name and the type of the
column where the fields will be written.
If the `raw=true` configuration is used for the output, then all the record is sent to
the output.

### How to create components

#### Filters

> An example code can be found at [./examples/filtering/filter.go](./examples/filtering/filter.go)

A filter must implement a `baker.Filter` interface:

```go
type Filter interface {
    Process(r Record, next func(Record))
    Stats() FilterStats
}
```

While `Stats()` returns a struct used to collect metrics (see the Metrics chapter), the `Process()`
function is where the filter logic is implemented.

Filters receive a `Record` and the `next()` function, that represents the next filtering function in
the filter chain. Also if the filter is the last of the chain, the `next()` function is valid
(in this case Baker will send the record to the output).

The filter can do whatever it likes with the `Record`, like adding or changing a value, dropping it
(not calling the `next()` function) or even splitting a `Record` calling `next()` multiple
times.

##### baker.FilterDesc

In case you plan to use a TOML configuration to build the Baker topology, the filter should also be
described using a `baker.FilterDesc` struct. In fact a list of `baker.FilterDesc` will be used to
populate `baker.Components`, which is an argument of `baker.NewConfigFromToml`. 

```go
type FilterDesc struct {
    Name   string
    New    func(FilterParams) (Filter, error)
    Config interface{}
    Help   string
}
```

###### Name

The `Name` of the filter must be unique as it will match the toml `[filter]` configuration:

```toml
[[filter]]
name = "FilterName"
    [filter.config]
    filterConfKey1 = "somevalue"
    filterConfKey2 = "someothervalue"
```

###### New

This is the constructor and returns the `baker.Filter` interface as well as a possible `error`.

###### Config

The filter can have its own configuration (as the `[filter.config]` fields above). The `Config` field
will be populated with a pointer to the configuration struct provided.

The `New` function will receive a `baker.FilterParams`. Its `DecodedConfig` will host the filter
configuration. It requires a type assertion to the filter configuration struct type to be used:

```go
func NewMyCustomFilter(cfg baker.FilterParams) (baker.Filter, error) {
    if cfg.DecodedConfig == nil {
        cfg.DecodedConfig = &MyCustomFilterConfig{}
    }
    dcfg := cfg.DecodedConfig.(*MyCustomFilterConfig)
}
```

###### Help

The help string can be used to build an help output (see the [help](./examples/help/) example).

#### Inputs

Baker inputs are defined using the `baker.InputDesc` struct.  
The `New` function must return a `baker.Input` component whose `Run` function
represents the hearth of the input.  
That function receives a channel where the data produced by the input must be
pushed in form of a `baker.Data`.  
The actual input data is a slice of bytes that will be parsed with `Record.Parse()`
by Baker before sending it to the filter chain.
The input can also add metadata to `baker.Data`. Metadata can be user-defined and
filters must know how to read and use metadata defined by the input.

#### Outputs

An output must implement the `Output` interface:

```go
type Output interface {
    Run(in <-chan OutputRecord, upch chan<- string)
    Stats() OutputStats
    CanShard() bool
}
```

The [sharding example output](./examples/sharding/output.go) is a simple implementation of
an output and can be used as source.

An output can have its own configuration and the `OutputRecord` records sent to the `Run`
function can be the complete record (into `OutputRecord.Record`) in case the `raw=true`
configuration has been used for the output or only a subset of fields will be sent into
`OutputRecord.Fields` if `fields=["field", "field", ...]` is used. In this latter case
the `OutputRecord.Fields` slice will have the same order of the `fields` configuration.

If more than one `procs` is used (the default value is 32), then each output process will
receive a subset of the records. The `OutputParams.Index` passed to the `New` function
identifies the output process and can be used to correcly handle parallelism.

Sharding (which is explained below) is strictly connected to the output component but
it's also transparent to it. An output will never know how the sharding is calculated,
but records with the same value on the field used to calculate sharding will be  always
sent to the output process with the same index (unless a broken sharding function is used).

The output also receives an upload channel where it can send strings to the uploader.
Those strings will likely be paths to something produced by the output (like files)
that the uploader must upload somewhere.

#### Uploads

As explained in the output paragragh, a string channel is used by the outputs to send messages
the the uploader. Those strings can represent, for example, file paths and those files
could be uploaded somewhere.

The uploader component is optional, if missing the string channel is simply ignored by Baker.

### How to create a '-help' command line option

The [./examples/help/](./examples/help/) folder contains a working example of
command that shows a generic help/usage message and also specific component
help messages when used with `-help <ComponentName>`

## Tuning parallelism

When testing Baker in staging environment, you may want to experiment with parallelism
options to try and obtain the best performance. Keep an eye on `htop` as the pipeline
runs: if CPUs aren't saturated, it means that Baker is running I/O-bound, so depending
on the input/output configuration you may want to increase the parallelism and squeeze
out more performance.

These are the options you can tune:

* Section `[filterchain]`:
  * `procs`: number of parallel threads running the filter chain (default: 16)
* Section `[output]`:
  * `procs`: number of parallel threads sending data to the output (default: 32)

## Sharding

Baker supports sharding of output data, depending on the value of specific fields
in each record. Sharding makes sense only for some specific output components,
so check each output component. A component that supports sharding must return `true`
from the function `CanShard()`.

To configure sharding, it's sufficient to create a `sharding` key in `[output]` section,
specifying the column on which the sharding must be executed.
`ShardingFuncs` in `baker.Components` must include a function for the selected field and the
function must return an index (`uint64`) for each possible value of the field. The index
is used to choose the target output procs for the records.
Since the sharding functions provide the capability to spread the records across different output
`procs` (parallel goroutines), it's clear that the `[output]` configuration must include a `procs`
value greater than 1 (or must avoid including it as the default value is 32).

### How to implement a sharding function

The [./examples/sharding/](./examples/sharding/) folder contains a working
example of an output that supports sharding and a `main()` configuration to
use it together with simple sharding functions.

## Stats

While running, Baker dumps stats on stdout every second. This is an example line:

```log
Stats: 1s[w:29425 r:29638] total[w:411300 r:454498] speed[w:27420 r:30299] errors[i:0 f:0 o:0]
```

The first bracket shows the number of records that were read (i.e. entered the pipeline)
and written (i.e. successfully exited the pipeline) in the last 1-second window. The second
bracket is total since the process was launched. The third bracket shows the average read/write
speed (records per second).

The fourth bracket shows the records that were discarded at some point during the records
because of errors:

* `i:` is the number of records that were discarded because an error occurred within
   the input component. Most of the time, this refers to validation issues.
* `f:` is the number of records that were discarded by the filters in the pipeline. Each
   filter can potentially discard records, and if that happens, it will be reported here.
* `o:` is the number of records that were discarded because of an error in the output
   component. Notice that output components should be resilient to transient network failures,
   and they abort the process in case of permanent configuration errors, so the number
   here reflects records that could not be permanently written because eg. validation
   issues. Eg. think of an output that expects a column to be in a specific format, and
   rejects records where that field is not in the expected format. A real-world example
   is empty columns that are not accepted by DynamoDB.

## Metrics

Baker can send metrics to Datadog during its execution. Each component provides some
metrics that are collected and regularly sent through the agent. To configure metrics
collection, use the following keys in the `[general]` section:

* `datadog` (bool): activate metrics collection (default: `false`)
* `datadog_prefix` (string): prefix for all collected metrics (default: `Baker`)
* `datadog_host` (string): hostname/port of the datadog agent to connect to
    (default: `127.0.0.1:8125`)

## Aborting (CTRL+C)

By design, Baker attempts a clean shutdown on CTRL+C (SIGINT). This means that it
will try to flush all records that it's currently processing and correctly
flush/close any output.

Depending on the configured pipeline and the requested parallelism, this could take
anything from a few seconds to a minute or so (modulo bugs...). Notice that improving
this requires active development, so feel free to open an issue if it's not working
fast enough fo you.

If you need to abort right away, you can use CTRL+\ (SIGQUIT).

## Baker test suite

You can run Baker tests as any other go project just executing `go test -v -race ./...`.  
The code also includes several benchmarks.
