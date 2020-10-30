---
title: "Pipeline configuration"
date: 2020-10-29
weight: 2
description: >
  How to configure Baker using TOML files
---

Baker is configured using a [TOML file](https://toml.io/en/), whose content is processed by the
[`NewConfigFromToml`](https://pkg.go.dev/github.com/AdRoll/baker#NewConfigFromToml) function.

The file has several sections, described below:

| Section       | Required   | Content                        |
|---------------|------------|--------------------------------|
| `[general]`   | false      | General configuration          |
| `[metrics]`   | false      | Metrics service configuration  |
| `[[user]]`    | false      | User-defined configurations    |
| `[input]`     | true       | Input component configuration  |
| `[[filter]]`  | false      | Filters configuration          |
| `[output]`    | true       | Output component configuration |
| `[upload]`    | false      | Upload component configuration |

### General configuration

The `[general]` section is used to configure the general behaviour of Baker.

| Key                    | Type   | Effect |
|------------------------|--------|--------|
| dont_validate_fields   | bool   | Reports whether records validation is skipped (by not calling Components.Validate) |

### Metrics configuration

The `[metrics]` section permits to configure the metrics host to use. Currently, only `datadog` is
supported.

See the dedicated page to learn how to configure DataDog metrics with Baker. (TODO: add link)

### User defined configurations

Using baker as library, one can profit of TOML parsing included in Baker to add custom configurations
that can be used in other parts of the code.
For details about how to use those configuration, read the dedicated page (TBD).

### Components configuration

Components sections are `[input]`, `[[filter]]`, `[output]` and `[upload]` and will contain a
`name = "<component name>"` line and an optional `config` subsection (like `[input.config]`)
to set specific configuration values to the selected component.

Components' specific configuration can be marked as required (within the component code). If a
required config is missing, baker won't start.

This is a minimalist baker configuration TOML, reading records from files (`List`), applying the
`ClauseFilter` filter (without specific configurations) and writing the output to `DynamoDB`,
with some specific options:

```toml
[input]
name="List"

    [input.config]
    files=["records.csv.gz"]

[[filter]]
name="ClauseFilter"

[output]
name="DynamoDB"
fields=["source","timestamp","user"]

    [output.config]
    regions=["us-west-2","us-east-1"]
    table="TestTableName"
    columns=["s:Source", "n:Timestamp", "s:User"]
```

`[input]` selects the input component, or where to read the records from.  
In this case, the List component is selected, which is a component that fetches CSV files from
a list of local or remote paths/URLs. `[input.config]` is where component-specific configurations
can be specified, and in this case we simply provide the files option to List.  
Notice that List would accept http:// or even s3:// URLs there in addition to local paths,  
and some more (run ./baker-bin -help List in the help example for more details).

`[[filter]]` In TOML syntax, the double brackets indicates an array of sections.  
This is where you declare the list of filters (i.e filter chain) to sequentially apply to your
records. As other components, each filter may be followed by a `[filter.config]` section.  
This is an example:

```toml
[[filter]]
name="filterA"

    [filter.config]
    foo = "bar"

[[filter]]
name="filterB"
```

`[output]` selects the output component; the output is where records that made it to the end of
the filter chain without being discarded end up. In this case, the `DynamoDB` output is selected,
and its configuration is specified in `[output.config]`.

The `fields` option in the `[output]` section selects which fields of the record will be send
to the output.  
In fact, most pipelines don't want to send the full records to the output, but they will select
a few important columns out of the many available columns.  
Notice that this is just a selection: it is up to the output component to decide how to
physically serialize those columns. For instance, the `DynamoDB` component requires the user
to specify an option called columns that specifies the name and the type of the column where
the fields will be written.

### Environment variables replacement

Baker supports environment variables replacement in the configuration file.

Use `${ENV_VAR_NAME}` or `$ENV_VAR_NAME` and the value in the file will be replaced at runtime.  
Note that if the variable doesn't exist, then an empty string will be used for replacement.
