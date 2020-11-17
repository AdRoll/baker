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
	components := baker.Components{
		Inputs:  input.All,
		Filters: filter.All,
		Outputs: output.All,
		Uploads: upload.All,
	}
	if err := baker.MainCLI(components); err != nil {
		log.Fatal(err)
	}
}
