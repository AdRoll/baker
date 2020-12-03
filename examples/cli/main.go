// CLI example illustrates how to build a Baker CLI program using baker.MainCLI
// Run without arguments to see the help. Run with options to explore them and
// run with the `example.toml` argument (the file in this folder) to execute the topology
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
	comp := baker.Components{
		Inputs:  input.All,
		Filters: filter.All,
		Outputs: output.All,
		Uploads: upload.All,
	}

	if err := baker.MainCLI(comp); err != nil {
		log.Fatal(err)
	}
}
