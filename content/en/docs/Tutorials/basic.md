---
title: "Basic: build a simple pipeline"
weight: 410
date: 2020-11-13
description: >
    A step-by-step tutorial to learn how to build a Baker pipeline using the included components
---
In this tutorial you'll learn how to create a Baker-based program to process a given dataset
(in CSV format), filter records based on your needs and save the result to a file.

The dataset we're going to use is an open dataset containing ratings on many
[Ramens](https://www.kaggle.com/residentmario/ramen-ratings).

Our goal will be to discard all ramens that have never been on a top-ten ranking, split the
results into multiple folders named after the ramens source countries, and upload the
resulting lists to S3.

## The dataset

The dataset file has 9 columns:

* **review_num**: the number of the review (higher numbers mean more recent reviews)
* **brand**: the name of the restaurant
* **variety**: the name of the recipe
* **style**: the type of the ramen (cup, pack, bowl, etc)
* **country**: self-explanatory
* **stars**: ratings stars (from 0 to 5)
* **top_ten**: whether the ramen has been included in a top-ten ranking

{{% alert title="Warning" color="warning" %}}
The [original CSV file](https://www.kaggle.com/residentmario/ramen-ratings) can't be immediately
used with Baker because:

* it includes a header row
* some fields have values with commas and thus are enclosed in double-quotes. Baker doesn't support it
* the file is uncompressed

For the purpose of this tutorial we've already prepared the final file for you and **it is
available for downloading [here](/tutorial-base-dataset.csv.gz).**
{{% /alert %}}

## The required components

* [`List`](/docs/components/inputs/list/): reads the input file from disk
* [`NotNull`](/docs/components/filters/notnull/): discards all ramens without a top-ten entry
* [`FileWriter`](/docs/components/outputs/filewriter/): saves the resulting file to disk
* [`S3`](/docs/components/uploads/s3/): uploads the file to S3

## Baker configuration

An essential thing to do is to create a configuration file for Baker, in
[TOML](https://github.com/toml-lang/toml) format, selecting the aforementioned components:

```toml
[input]
name = "List"

    [input.config]
    Files = ["/tmp/db.csv.gz"] # put the file wherever you like

[[filter]]
name = "NotNull"

    [filter.config]
    Fields = ["top_ten"] # discard all records with an empty top_ten field

[output]
name = "FileWriter"
procs = 1 # With our PathString, FileWriter doesn't support concurrency
fields = ["country"]

    [output.config]
    PathString = "/tmp/out/{{.Field0}}/ramens.csv.gz"

[upload]
name="S3"

    [upload.config]
    Region = "us-east-1"
    Bucket = "myBucket"
    Prefix = "ramens/"
    StagingPath = "/tmp/staging/"
    SourceBasePath = "/tmp/out/"
    Interval = "60s"
    ExitOnError = true
```

## Create the program

Baker is a Go library. To use it, it is required to create a Go `main()` function,
[define a `baker.Components`](/docs/how-tos/baker_components/) object and pass it to
[`baker.MainCLI()`](https://pkg.go.dev/github.com/AdRoll/baker#MainCLI):

```go
package main

import (
	"log"

    "github.com/AdRoll/baker"
)

func main() {
    components := baker.Components{/* define components */}
    if err := baker.MainCLI(components); err != nil {
		log.Fatal(err)
	}
}
```

### Define baker.Components

Our `baker.Components` implementation must:

* include the components we need to use
* define `FieldByName` and `FieldName` to map our dataset

The simplest way to add the components to Baker is just to add all available components:

```go
components := baker.Components{
    Inputs:      input.All,
    Filters:     filter.All,
    Outputs:     output.All,
    Uploads:     upload.All,
    /* ... */
}
```

As for the mapping functions, we need to map from name to `FieldIndex` and vice-versa:

```go
var fields = []string{
	"review_num",
	"brand",
	"variety",
	"style",
	"country",
	"stars",
	"top_ten",
}

var fieldsByName = map[string]baker.FieldIndex{
	"review_num": 0,
	"brand":      1,
	"variety":    2,
	"style":      3,
	"country":    4,
	"stars":      5,
	"top_ten":    6,
}

func fieldByName(name string) (idx baker.FieldIndex, ok bool) {
	idx, ok = fieldsByName[name]
	return idx, ok
}

func fieldName(idx baker.FieldIndex) string {
	return fields[idx]
}

components := baker.Components{
    /* ... */
    FieldByName: fieldByName,
    FieldName:   fieldName,
}
```

The complete program is available in the
[`tutorials/` folder](https://github.com/AdRoll/baker/blob/main/tutorials/basic/main.go) in
the Baker repository.

## Run the program

Once the code and the configuration file are ready, we can run the topology:

```sh
$ go build -o myProgram ./main.go 
# Test it works as expected
$ ./myProgram -help
# run the topology
$ ./myProgram topology.toml
```

Among the messages that Baker prints on stdout, the stats messages are particularly interesting:

```sh
Stats: 1s[w:0 r:0] total[w:41 r:2584 u:11] speed[w:20 r:1292] errors[p:0 i:0 f:2543 o:0 u:0]
```

Take a look at the [dedicated page](/docs/how-tos/read_stats/) to learn how to read the values.

## Verify the result

The resulting files are splitted into multiple folders, one for each country, and then uploaded.

The [`S3`](/docs/components/uploads/s3/) upload removes the files once uploaded, so you'll find
only the folders skeleton in the output destination folder:

```sh
~ ls --tree -l /tmp/out
drwxrwxr-x   - username 16 Nov 11:43 /tmp/out
drwxrwxr-x   - username 16 Nov 11:43 ├── China
drwxrwxr-x   - username 16 Nov 11:43 ├── Hong Kong
drwxrwxr-x   - username 16 Nov 11:43 ├── Indonesia
drwxrwxr-x   - username 16 Nov 11:43 ├── Japan
drwxrwxr-x   - username 16 Nov 11:43 ├── Malaysia
drwxrwxr-x   - username 16 Nov 11:43 ├── Myanmar
drwxrwxr-x   - username 16 Nov 11:43 ├── Singapore
drwxrwxr-x   - username 16 Nov 11:43 ├── South Korea
drwxrwxr-x   - username 16 Nov 11:43 ├── Taiwan
drwxrwxr-x   - username 16 Nov 11:43 ├── Thailand
drwxrwxr-x   - username 16 Nov 11:43 └── USA
```

The uploaded files have been uploaded to S3:

```sh
~ aws s3 ls --recursive s3://myBucket/ramens/
2020-11-16 11:43:59        115 ramens/China/ramens.csv.gz
2020-11-16 11:43:59         83 ramens/Hong Kong/ramens.csv.gz
2020-11-16 11:43:59        223 ramens/Indonesia/ramens.csv.gz
2020-11-16 11:43:59        236 ramens/Japan/ramens.csv.gz
2020-11-16 11:43:59        240 ramens/Malaysia/ramens.csv.gz
2020-11-16 11:43:59         99 ramens/Myanmar/ramens.csv.gz
2020-11-16 11:43:59        219 ramens/Singapore/ramens.csv.gz
2020-11-16 11:43:59        265 ramens/South Korea/ramens.csv.gz
2020-11-16 11:43:59        159 ramens/Taiwan/ramens.csv.gz
2020-11-16 11:43:59        181 ramens/Thailand/ramens.csv.gz
2020-11-16 11:43:59         94 ramens/USA/ramens.csv.gz
```

## Conclusion

This is it for this basic tutorial. You have learned:

* how to create a simple Baker program to process a CSV dataset with minimal filtering and upload it to S3
* how to create the Baker TOML configuration file
* how to execute the program and verify the result

You can now improve your Baker knowledge by taking a look at the [other tutorials](/docs/tutorials/)
and learning more [advanced topics](/docs/how-tos/).
