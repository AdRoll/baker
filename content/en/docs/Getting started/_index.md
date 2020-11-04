---
title: "Getting started"
linkTitle: "Getting Started"
weight: 100
---

Baker is a Go library and it must be included into your Go program.

To create a new Go project using Go modules and adding Baker:

```sh
mkdir myProject
cd myProject
go mod init github.com/myUser/myProject
go get github.com/adroll/baker
```

If you are adding Baker to a project already configured to use Go modules, just type:

```sh
cd myProject
go get github.com/adroll/baker
```

Read [Baker Core concepts](/docs/core-concepts/) to know how Baker works or go to the
[API reference](https://pkg.go.dev/github.com/AdRoll/baker) to learn how to use it in your code.
