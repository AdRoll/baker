package main

import (
	"flag"
	"os"
	"strings"
	"text/template"

	"github.com/AdRoll/baker"
)

var hextable [256]uint8

func init() {
	for i := range hextable {
		hextable[i] = 0xFF
	}
	for i := 'a'; i <= 'f'; i++ {
		hextable[i] = uint8(i) - 'a' + 10
	}
	for i := 'A'; i <= 'F'; i++ {
		hextable[i] = uint8(i) - 'A' + 10
	}
	for i := '0'; i <= '9'; i++ {
		hextable[i] = uint8(i) - '0'
	}
}

func uuidToInt(uuid []byte) uint64 {
	n := len(uuid)
	hash := uint64(5381)

	for i := 0; i < n; i++ {
		v := hextable[uuid[i]]
		v *= 16
		i++

		c := hextable[uuid[i]]
		v += c
		hash = hash*33 + uint64(v)
	}

	return hash
}

var programUsageTemplate = template.Must(template.New("Program usage").Parse(`
Baker version: {{ .Build }}

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
		Build      string
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
		Build:      build,
		ExecName:   os.Args[0],
		Defaults:   defaultsBuilder.String(),
		Components: components,
	}); err != nil {
		panic(err)
	}
}
