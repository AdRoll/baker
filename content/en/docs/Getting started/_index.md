---
title: "Getting started"
linkTitle: "Getting Started"
weight: 100
description: Quick Baker setup instructions
---

Looking to use Baker and start building pipelines now? Great, let's see what you need.

Baker is written in Go. To use it you need to import the Baker module into your program.  
In this page we describe the simplest way to use Baker. At the end of it, we recommend reading the
[Baker Core Concepts](/docs/core-concepts/) and then have a deep dive in the
[How-to pages](/docs/how-tos/).

## Add Baker...

### ...to a brand-new project

To create a new Go project (using Go modules) and add Baker, these are the suggested steps:

```sh
mkdir myProject
cd myProject
go mod init github.com/myUser/myProject
go get github.com/AdRoll/baker
```

### ...to an existing project

If you are adding Baker to a project already configured to use Go modules, just type:

```sh
cd myProject
go get github.com/AdRoll/baker
```

## Build and run Baker

Once Baker has been added to the project, let's see how to use it, with a minimalistic example.

This code comes from the [cli](https://github.com/AdRoll/baker/tree/main/examples/cli)
example. You can find more examples in the
[`examples/` folder]((https://github.com/AdRoll/baker/tree/main/examples/cli)) of the project.

```go
package main

import (
    "log"

    "github.com/AdRoll/baker"
    "github.com/AdRoll/baker/filter"
    "github.com/AdRoll/baker/input"
    "github.com/AdRoll/baker/output"
    "github.com/AdRoll/baker/upload"
)

func main() {
    // Add all available components
    comp := baker.Components{
        Inputs:  input.All,
        Filters: filter.All,
        Outputs: output.All,
        Uploads: upload.All,
    }

    // run Baker
    if err := baker.MainCLI(comp); err != nil {
        log.Fatal(err)
    }
}
```

To create the binary, just build it:

```sh
cd myProject
go build -o baker-example .
```

### Configuration

In the example above we use [`baker.MainCLI`](https://pkg.go.dev/github.com/AdRoll/baker#MainCLI),
an utility function that hides a lot of commonly used setup and requires a
[TOML file](https://github.com/toml-lang/toml) as first command line parameter.

The TOML file should the Baker configuration that suits your needs.  
For details about how to configure Baker,
[read the dedicated page](http://localhost:1313/docs/how-tos/pipeline_configuration/).

A simple example, in this case coming from the
[basic](https://github.com/AdRoll/baker/tree/main/examples/basic/main.go) example, is the following:

```toml
[fields]
names=["timestamp", "source", "target"]

[input]
name = "List"

	[input.config]
	files=["testdata/input.csv.gz"]

[[filter]]
name="ReplaceFields"

	[filter.config]
	ReplaceFields=["replaced", "timestamp"]

[output]
name = "FileWriter"
procs=1

	[output.config]
	PathString="./_out/output.csv.gz"
```

### Run the program

Running the program is as simple as it sounds, at this point:

```sh
$ ./baker-example /path/to/configuration.toml

INFO[0000] Initializing                                  fn=NewFileWriter idx=0
INFO[0000] FileWriter ready to log                       idx=0
INFO[0000] begin reading                                 f=compressedInput.parseFile fn=testdata/input.csv.gz
INFO[0000] end                                           f=compressedInput.parseFile fn=testdata/input.csv.gz
INFO[0000] terminating                                   f=List.Run
INFO[0000] Rotating                                      current= idx=0
INFO[0000] Rotated                                       current= idx=0
INFO[0000] FileWriter Terminating                        idx=0
INFO[0000] fileWorker closing                            idx=0
```

## Next steps

Do you want to know more about Baker?  
You can read [Baker Core concepts](/docs/core-concepts/) or, if you prefer to jump straight into
the code, you can browse the [API reference](https://pkg.go.dev/github.com/AdRoll/baker).

More detailed examples can be found in the [How-tos section](/docs/how-tos/) and the available
components are documented in the [components pages](/docs/components/).
