---
title: "Create a custom upload component"
date: 2020-11-11
weight: 750
---
The last (optional) component of a Baker pipeline is the Upload, whose job is to, precisely,
upload somewhere what the output component produces.

To create an upload component and make it available to Baker, one must:

* Implement the [Upload](https://pkg.go.dev/github.com/AdRoll/baker#Upload) interface
* Fill an [`UploadDesc`](https://pkg.go.dev/github.com/AdRoll/baker#UploadDesc) structure and
register it within Baker via [`Components`](https://pkg.go.dev/github.com/AdRoll/baker#Components).

## The Upload interface

```go
type Upload interface {
	Run(upch <-chan string) error
	Stop()
	Stats() UploadStats
}
```

The [Upload interface](https://pkg.go.dev/github.com/AdRoll/baker#Upload) must be implemented when
creating a new Upload component.

The `Run` function implements the component logic and receives a channel where the output sends what
it produces (most probably file paths).

Currently, only the [S3](https://github.com/AdRoll/baker/blob/main/upload/s3.go) component exists
and it expects to receive in the channel the path to the files produced by an output.

## UploadDesc

```go
var MyUploadDesc = baker.UploadDesc{
	Name:   "MyUpload",
	New:    NewMyUpload,
	Config: &MyUploadConfig{},
	Help:   "High-level description of MyUpload",
}
```

This object has a `Name`, that is used in the Baker configuration file to identify the upload,
a constructor-like function (`New`), a config object (where the parsed upload configuration from the
TOML file is stored) and a help text that must help the users to use the component and its
configuration parameters.

### Upload constructor-like function

The `New` key in the `UploadDesc` object should be set to a function returning an Upload.

The function receives a [UploadParams](https://pkg.go.dev/github.com/AdRoll/baker#UploadParams)
object and returns an instance of [Upload](https://pkg.go.dev/github.com/AdRoll/baker#Upload).

The function should verify the configuration params into `UploadParams.DecodedConfig` and initialize
the component accordingly.

### The upload configuration and help

The upload configuration object (`MyUploadConfig` in the previous example) must export all
configuration parameters that the user can set in the TOML topology file.

Each key in the struct must include a `help` string tag and a `required` boolean tag, both are
mandatory.

The former helps the user to understand the possible values of the field, the latter tells Baker
whether to refuse a missing configuration param.

## The shared channel

What the upload component receives in the string channel is, indeed, a string and, at the moment,
the only upload implementation (S3) expects to find there the path to local files.

The S3 upload removes those files once uploaded, but there isn't a golden rule for what to do with
them. This is up to the upload component and should be chosen wisely and documented extensively.

At the same time, it is not an obligation for the string sent into the channel to represent
a local file path and thus it is also possible that an upload component is not compatible with an
output component (depending on what is sent to the channel and what is expected).

Again, as there is not a golden rule about what is sent to the channel, the best thing to do for
both the output and upload components is to create good documentation to inform the users of their
behaviors.
