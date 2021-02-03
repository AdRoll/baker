// basic example illustrates how to build a very simple baker-based program with just
// input and output components
package main

import (
	"log"
	"strings"

	"github.com/AdRoll/baker"
	"github.com/AdRoll/baker/filter"
	"github.com/AdRoll/baker/input"
	"github.com/AdRoll/baker/output"
)

func main() {
	toml := `
[fields]
	names=["timestamp", "source", "target"]

[validation]
	timestamp="^value[0-9]+$"
	target="value3"

[input]
name = "List"

	[input.config]
	files=["testdata/input.csv.zst"]

[output]
name = "FileWriter"
procs=1

	[output.config]
	PathString="_out/output.csv.gz"
`
	c := baker.Components{
		Inputs:  input.All,
		Outputs: output.All,
		Filters: filter.All,
	}
	cfg, err := baker.NewConfigFromToml(strings.NewReader(toml), c)
	if err != nil {
		log.Fatal(err)
	}

	if err := baker.Main(cfg); err != nil {
		log.Fatal(err)
	}
}
