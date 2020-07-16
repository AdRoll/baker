// help is a simple example illustrating how to build a baker command with the ability
// to show an help output for common usage and component-specific instructions.
// A wrong invocation (no toml argument) or the -h/-help argument shows the main usage help.
// Using -help together with an available component, shows its help. `-help *` shows the
// help for all available components
package main

import (
	"flag"
	"fmt"
	"html/template"
	"os"
	"strings"

	"github.com/AdRoll/baker"
	"github.com/AdRoll/baker/filter"
	"github.com/AdRoll/baker/input"
	"github.com/AdRoll/baker/output"
)

var flagHelpConfig = flag.String("help", "", "show help for a `component` (use '*' to dump all)")

func main() {

	flag.Usage = displayProgramUsage
	flag.Parse()

	if *flagHelpConfig != "" {
		baker.PrintHelp(os.Stderr, *flagHelpConfig, components)
		return
	}

	// If the toml configuration file isn't provided as argument, print usage and exit
	if len(flag.Args()) < 1 {
		flag.Usage()
		os.Exit(1)
	}

	// Do stuff, then...
	fmt.Println("Bye!")
}

var components = baker.Components{
	Inputs:  input.All,
	Filters: filter.All,
	Outputs: output.All,
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
		ExecName:   os.Args[0],
		Defaults:   defaultsBuilder.String(),
		Components: components,
	}); err != nil {
		panic(err)
	}
}
