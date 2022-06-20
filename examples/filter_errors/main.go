// filtering example illustrates how to create a filter component.
// It takes the basic example adding a "Lazy" filter definition
package main

import (
	_ "embed"
	"log"
	"strings"

	"github.com/AdRoll/baker"
	"github.com/AdRoll/baker/filter"
	"github.com/AdRoll/baker/filter_error_handler"
	"github.com/AdRoll/baker/input"
	"github.com/AdRoll/baker/output"
)

//go:embed topo.toml
var toml string

func main() {
	c := baker.Components{
		Inputs:              input.All,
		Filters:             filter.All,
		Outputs:             output.All,
		FilterErrorHandlers: filter_error_handler.All,
	}
	cfg, err := baker.NewConfigFromToml(strings.NewReader(toml), c)
	if err != nil {
		log.Fatal(err)
	}

	if err := baker.Main(cfg); err != nil {
		log.Fatal(err)
	}
}
