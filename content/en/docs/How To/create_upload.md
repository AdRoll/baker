---
title: "Create a custom upload component"
date: 2020-11-11
weight: 750
---
The last (optional) component of a Baker pipeline is the Upload, whose job is to, precisely,
upload local files produced by the output component.

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

The function receives an [UploadParams](https://pkg.go.dev/github.com/AdRoll/baker#UploadParams)
object and returns an instance of [Upload](https://pkg.go.dev/github.com/AdRoll/baker#Upload).

It should verify the configuration, accessed via `UploadParams.DecodedConfig` and initialize
the component accordingly.

### Upload configuration and help

The upload configuration object (`MyUploadConfig` in the previous example) must export all
configuration parameters that the user can set in the TOML topology file.

Each field in the struct must include a `help` string tag (mandatory) and a `required` boolean tag
(default to `false`).

All these parameters appear in the generated help. `help` should describe the parameter role and/or
its possible values, `required` informs Baker it should refuse configurations in which that field
is not defined.

## The files to upload

Through the channel, the upload receives from the output paths to local files that it must upload.

The only upload existing at the moment, S3, removes those files once uploaded, but there isn't a
golden rule for what to do with them. This is up to the upload component and should be chosen
wisely and documented extensively.
