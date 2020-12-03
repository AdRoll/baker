---
title: "FileWriter"
weight: 20
date: 2020-12-03
---
## Output *FileWriter*

### Overview
This is a *raw* output, for each record it receives a buffer containing the serialized record, plus a list holding a set of fields (`output.fields` in TOML).


This output writes the records into compressed files in a directory.  

Files will be compressed using Gzip or Zstandard based on the filename extension in PathString.  

The file names can contain placeholders that are populated by the output (see the keys help below).  

When the special {{.  
Field0}} placeholder is used, then the user must specify the field name to
use for replacement in the fields configuration list.  

The value of that field, extracted from each record, is used as replacement and, moreover, this
also means that each created file will contain only records with that same value for the field.  

Note that, with this option, the FileWriter creates as many workers as the different values
of the field, and each one of these workers concurrently writes to a different file.  



### Configuration

Keys available in the `[output.config]` section:

|Name|Type|Default|Required|Description|
|----|:--:|:-----:|:------:|-----------|
| PathString| string| ""| false| Template to describe location of the output directory: supports .Year, .Month, .Day and .Rotation. Also .Field0 if a field name has been specified in the output's fields list.|
| RotateInterval| duration| 60s| false| Time after which data will be rotated. If -1, it will not rotate until the end.|
| ZstdCompressionLevel| int| 3| false| zstd compression level, ranging from 1 (best speed) to 19 (best compression).|
| ZstdWindowLog| int| 0| false| Enable zstd long distance matching. Increase memory usage for both compressor/decompressor. If more than 27 the decompressor requires special treatment. 0:disabled.|

