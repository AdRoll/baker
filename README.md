# Baker

> This document is still WIP

Baker is a library to be used to create configurable pipelines for processing log files.  
It can read logfiles from different sources (S3, Kinesis, etc.), process them through
custom filters and send them to some output (like DynamoDB).

Baker is fully parallel and maximizes usage of both CPU-bound and I/O bound pipelines.
On a AWS c3.4x instance, it can run a simple pipeline (with little processing of
records) achieving 30k writes per seconds to DynamoDB in 4 regions, using ~1GB of RAM in
total, and 500% of CPU (so with room for scaling even further if required).

## Pipelines

A pipeline is the configured set of operations that Baker performs during its execution.

It is defined by:

* One input component, defining where to fetch log files from.
* Zero or more filters, which are functions that can modify records (changing fields,
  dropping them, or even "splitting" them into multiple records).
* One output component, defining where to send the filtered records to (and which
  columns).
* One optional upload component, defining where to send files produced by the output
  component (if any).  The only currently supported destination is an S3 bucket+prefix.

Notice that there are two main usage scenarios for Baker:

 1. Baker as a batch processor. In this case, Baker will go through all the records
    that are fed by the input component, process them as quickly as possible, and exit.
 2. Baker as a daemon. In this case, baker will never exit; it will keep waiting for
    incoming records from the input component (e.g.: Kinesis), process them and send
    them to the output.

Selecting between scenario 1 or scenario 2 is just a matter of configuring the pipeline;
in fact, it is the input component that drives the scenario. If the input component
exits at some point, Baker will flush the pipeline and exit as well; if instead the
input component is endless, Baker will never exit and thus behave like a daemon.

## Usage

Baker uses a `baker.Config` struct to know what to do. The configuration can be either created
manually or importing a toml file `baker.NewConfigFromToml()`. This
function requires a `baker.Components` object including all available components.  
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

Components (inputs, filters, outputs and uploads) can be user-defined or those
provided by baker can be used. They can also be merged in a single slice to use both.

## How to build a Baker executable

This paragraph shows how to build a baker-based executable. Goals of this example are:

* Create a `main()` function that uses Baker
* Read Baker configuration from a TOML file
* Provide Baker a list of components, including custom components
* ...

## TOML Configuration files

In case you want to configure baker starting from a toml file then parsed with
`baker.NewConfigFromToml`, this is a minimalist Baker pipeline that reads a logfile from the disk,
updates its timestamp field with a "Timestamp" filter and pushes it to DynamoDB:

```toml
[input]
name="List"

    [input.config]
    files=["./mylog.csv.gz"]

[[filter]]
name="Timestamp"

[output]
name="DynamoDB"
fields=["cookie","timestamp","recommended_products"]


    [output.config]
    regions=["us-west-2","us-east-1"]
    table="TestTableName"
    columns=["s:AdvCookie", "n:Timestamp", "s:Products"]
```

`[input]` selects the input component, or where to read the logfiles from. In this case,
the `List` component is selected, which is a component that fetches logs from a list of
local or remote paths/URLs. `[input.config]` is where component-specific configurations
can be specified, and in this case we simply provide the `files` option to `List`. Notice
that `List` would accept `http://` or even `s3://` URLs there in addition to local paths,
and some more (run `./baker -help List` for more details).

`[[filter]]` selects which filters to run. In TOML syntax, the double brackets indicate
an array of sections, so it basically means that you can define many different `[[filter]]`
sections, one for each filter that you wish to run. In this example, only the `Timestamp`
filter is selected. This configuration file doesn't specifiy any option for this filter;
if needed, those options would go to a `[filter.config]` subsection.

`[output]` selects the output component, that is where the records are sent to.
In this case, the `DynamoDB` component is selected, and its configuration is specified
in `[output.config]`.

The `fields` option in the `[output]` section selects which fields of the record will be
send to the output. In fact, most pipelines don't want to send the full records to the
output, but they will select a few important columns out of the many available columns.
Notice that this is just a selection: it is up to the output component to decide how to
physically serialize those columns. For instance, the `DynamoDB` component requires the
user to specify an option called `columns` that specifies the name and the type of the
column where the fields will be written.

### How to create components

#### Filters

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

The filter could have a configuration (as the `[filter.config]` fields above). The `Config` field
will be populated to a pointer to that configuration struct.

The `New` function will receive a `baker.FilterParams`. Its `DecodedConfig` will host the filter
configuration. It requires a type assertion to the filter configuration struct type to be used:

###### Help

The help string can be used to build an help output (see the "Help" paragraph).

##### Example

This is an example of filter, including its toml configuration.  
The filter gets via configuration a name of the record field and a possible value. All records with
different values for that field are filtered out.

```go
/* TOML configuration for the field
[[filter]]
name = "MyFilter"
    [filter.config]
    FieldName = "Company"
    AcceptedValue = "NextRoll"
*/
var MyFilterDesc = baker.FilterDesc{
    Name:   "MyFilter",
    New:    NewMyFilter,
    Config: &MyFilterConfig{},
    Help:   `Drops lines with invalid value for the given field`,
}

type MyFilterConfig struct {
    FieldName     string `help:"The name of the field to filter on"`
    AcceptedValue string `help:"The accepted value for the filtered field"`
}

type MyFilter struct {
    numProcessedLines int64
    numFilteredLines  int64
    cfg               *MyFilterConfig
    idx               baker.FieldIndex
}

func NewMyFilter(cfg baker.FilterParams) (baker.Filter, error) {
    if cfg.DecodedConfig == nil {
        cfg.DecodedConfig = &MyFilterConfig{}
    }
    dcfg := cfg.DecodedConfig.(*MyFilterConfig)
    idx, ok := cfg.FieldByName(dcfg.FieldName)
    if !ok { /* Return an error */ }
    return &MyFilter{cfg: dcfg, idx: idx}
}

func (f *MyFilter) Stats() baker.FilterStats {
    return baker.FilterStats{
        NumProcessedLines: atomic.LoadInt64(&f.numProcessedLines),
        NumFilteredLines:  atomic.LoadInt64(&f.numFilteredLines),
    }
}

func (f *MyFilter) Process(l baker.Record, next func(baker.Record)) {
    atomic.AddInt64(&onp.numProcessedLines, 1)

    if !bytes.Equal(l.Get(f.idx), []byte(f.cfg.AcceptedValue)) {
        atomic.AddInt64(&onp.numProcessedLines, 1)
        // Filter out the record not calling next()
        return
    }
    // Call next filter in the filter chain
    next(l)
}
```

#### Inputs

TODO

#### Outputs

TODO

#### Uploads

TODO

### How to create a '-help' command line option

The `PrintHelp` function can be used to provide a command line option to get components' help.

The help message includes the generic description of the component (provided by the `Help` attribute
of the `OutputDesc` struct) as well as the help messages for all component's configuration keys.

In case `*` is used as component name, the function shows the help messages for all known components.

An example of usage is:

```go
var flagPrintHelp = flag.String("help", "", "show help for a `component` (use '*' to dump all)")
flag.Parse()
if *flagPrintHelp != "" {
    comp := baker.Components{
        Inputs:  input.AllInputs(),
        Filters: filter.AllFilters(),
        Outputs: output.AllOutputs(),
    }
    PrintHelp(os.Stderr, *flagPrintHelp, comp)
    return
}
```

An example of help output is the following:

```
$ ./baker-bin -help DynamoDB
=============================================
Output: DynamoDB
=============================================
This output writes the filtered records to DynamoDB. It must be
configured specifying the region, the table name, and the columns
to write.
Columns are specified using the syntax "t:name" where "t"
is the type of the data, and "name" is the name of column. Supported
types are: "n" - integers; "s" - strings.
The first column (and field) must be the primary key.
Keys available in the [output.config] section:
Name               | Type               | Default                    | Help
----------------------------------------------------------------------------------------------------
Regions            | array of strings   | us-west-2                  | DynamoDB regions to connect to
Table              | string             |                            | Name of the table to modify
Columns            | array of strings   |                            | Table columns that correspond to each of
                   |                    |                            |   the fields being written
FlushInterval      | duration           | 1s                         | Interval at which flush the data to
                   |                    |                            |   DynamoDB even if we have not reached 25
                   |                    |                            |   records
MaxWritesPerSec    | int                | 0                          | Maximum number of writes per second that
                   |                    |                            |   DynamoDB can accept (0 for unlimited)
MaxBackoff         | duration           | 2m                         | Maximum retry/backoff time in case of
                   |                    |                            |   errors before giving up
----------------------------------------------------------------------------------------------------
```

## Tuning parallelism

When testing Baker in staging environment, you may want to experiment with parallelism
options to try and obtain the best performance. Keep an eye on `htop` as the pipeline
runs: if CPUs aren't saturated, it means that baker is running I/O-bound, so depending
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
for the function `CanShard()`.

To configure sharding, it's sufficient to create a key `sharding` in section `[output]`,
specifying the column on which the sharding must be executed.
`ShardingFuncs` in `baker.Components` must include a function for the selected field and the
function must return an index (`uint64`) for each possible value of the field. The index
is used to choose the target output procs for the records.

### How to implement a sharding function

TODO

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
* `datadog_prefix` (string): prefix for all collected metrics (default: `baker`)
* `datadog_host` (string): hostname/port of the datadog agent to connect to
    (default: `127.0.0.1:8125`)

## Aborting (CTRL+C)

By design, baker attempts a clean shutdown on CTRL+C (SIGINT). This means that it
will try to flush all records that it's currently processing and correctly
flush/close any output.

Depending on the configured pipeline and the requested parallelism, this could take
anything from a few seconds to a minute or so (modulo bugs...). Notice that improving
this requires active development, so feel free to open an issue if it's not working
fast enough fo you.

If you need to abort right away, you can use CTRL+\ (SIGQUIT).

## Baker test suites

You can run baker tests by executing:

```sh
GOFLAGS=-mod=vendor go test -v -race ./...
```
