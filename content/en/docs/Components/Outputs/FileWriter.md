---
title: "FileWriter"
weight: 31
date: 2021-08-11
---
{{% pageinfo color="primary" %}}

**Read the [API documentation &raquo;](https://pkg.go.dev/github.com/AdRoll/baker/output#FileWriter)**
{{% /pageinfo %}}

## Output *FileWriter*

### Overview
This is a *raw* output, for each record it receives a buffer containing the serialized record, plus a list holding a set of fields (`output.fields` in TOML).


This output writes serialized records into compressed files, gzip (.gz) or zstd
(.zst) depending on the file extension in PathString.

Generated files may be rotated if RotateInterval is set. PathString is used to
control the name of the generated files, it may contain placeholders. These
placeholders are evaluated each time a file is created, that is upon creation
of the output or everytime a rotation takes place.

Supported placeholders:
 - {{.Year}}      year at file creation, 4 digits (YYYY)
 - {{.Month}}     month number at file creation, 2 digits (MM)
 - {{.Day}}       day of the month at file creation, 2 digits (DD)
 - {{.Hour}}      hour at file creation in 24h format, 2 digits (HH)
 - {{.Minute}}    minute at file creation, 2 digits (MM)
 - {{.Second}}    second at file creation, 2 digits (SS)
 - {{.Index}}     index of the current output process (see [output.procs]), 4 digits long
 - {{.UUID}}      per-worker random UUID (v4 UUID), 36 chars long
 - {{.Rotation}}  rotation count, 6 digits long
 - {{.Field0}}    value of the first field provided in [output.fields] (only if present).
 
When choosing configuration values for your FileWriter, it's important to keep in mind
the following rules:

 1. A file should only ever be accessed by a single worker at a time.

If you use multiple output processes, you should use {{.Index}} or {{.UUID}} 
so that generated filenames are guaranteed to be different for each workers.

 2. Rotation should never generate the same path twice.
 
To avoid a file to be overwritten by its successor in the rotation, you should ensure
that 2 files generated at a distance of RotateInterval will have different filenames.
To ensure filenames are different, you should set RotateInterval to a duration that 
exceeds that of the time-based placeholder with the shortest span.

For example, the following is correct since it's the generated path is guaranteed to
be unique at each rotation:

    PathString = "/path/to/file-{{.Hour}}-{{.Minute}}.log.gz" 
    RotateInterval = 5m

However, this is not correct, since successive generations may generate the exact same 
path:

    PathString = "/path/to/file-{{.Hour}}-{{.Minute}}.log.gz" 
    RotateInterval = 1s

 3. Only use {{.Field0}} if you trust the records you consume.

By using {{.Field0}} the files produces will have a path containing whatever value
is found. It could contain characters that are not valid to appear in a path. That also
means that the number of files (and workers) depend on the cardinality of that field.

### Configuration

Keys available in the `[output.config]` section:

|Name|Type|Default|Required|Description|
|----|:--:|:-----:|:------:|-----------|
| PathString| string| ""| false| Template describing names of the generated files. See top-level documentation for supported placeholders..|
| RotateInterval| duration| 60s| false| Time interval between 2 successive file rotations. -1 disabled rotation.|
| ZstdCompressionLevel| int| 3| false| Zstd compression level, ranging from 1 (best speed) to 19 (best compression).|
| ZstdWindowLog| int| 0| false| Enable zstd long distance matching. Increase memory usage for both compressor/decompressor. If more than 27 the decompressor requires special treatment. 0:disabled.|

