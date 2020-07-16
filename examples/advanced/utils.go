package main

import (
	"flag"
	"hash/fnv"
	"os"
	"strings"
	"text/template"

	"github.com/AdRoll/baker"
)

func simpleHash(r baker.Record, idx baker.FieldIndex) uint64 {
	f := fnv.New64()
	f.Write(r.Get(idx))
	return f.Sum64()
}

var programUsageTemplate = template.Must(template.New("Program usage").Parse(`
Usage: {{ .ExecName }} [options] TOPOLOGY

TOPOLOGY must be a pathname to a TOML file describing the topology to create.

Options:
{{ .Defaults }}

Available inputs:
{{ range .Components.Inputs }}
  {{ .Name }}{{ end }}

Available filters:
{{ range .Components.Filters }}
  {{ .Name }}{{ end }}

Available outputs:
{{ range .Components.Outputs }}
  {{ .Name }}{{ end }}

Available uploads:
{{ range .Components.Uploads }}
  {{ .Name }}{{ end }}
`))

func displayProgramUsage() {
	// Structure program usage sections
	type programUsage struct {
		ExecName   string
		Defaults   string
		Components baker.Components
	}

	// Capture command argument defaults
	var defaultsBuilder strings.Builder
	flag.CommandLine.SetOutput(&defaultsBuilder)
	flag.PrintDefaults()

	// Inject program usage data into message template
	if err := programUsageTemplate.Execute(os.Stderr, &programUsage{
		ExecName:   os.Args[0],
		Defaults:   defaultsBuilder.String(),
		Components: components,
	}); err != nil {
		panic(err)
	}
}
