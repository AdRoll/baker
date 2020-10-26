// metrics example illustrates how to implement and plug a metrics interface
// to Baker.
package main

import (
	"log"
	"strings"

	"github.com/AdRoll/baker"
	"github.com/AdRoll/baker/input"
	"github.com/AdRoll/baker/output"
)

// Some example fields
const (
	Timestamp baker.FieldIndex = 0
	Source    baker.FieldIndex = 1
	Target    baker.FieldIndex = 2
)

var fields = map[string]baker.FieldIndex{
	"timestamp": Timestamp,
	"source":    Source,
	"target":    Target,
}

func fieldByName(key string) (baker.FieldIndex, bool) {
	idx, ok := fields[key]
	return idx, ok
}

func main() {
	toml := `
[metrics]
name="Foobar"

	[metrics.config]
	host="metrics.foobar.com"
	port=8080

[input]
name = "List"

	[input.config]
	files=["testdata/input.csv.zst"]

[output]
name = "FileWriter"
procs=1

	[output.config]
	PathString="./_out/output.csv.gz"
`
	c := baker.Components{
		Inputs:      input.All,
		Outputs:     output.All,
		FieldByName: fieldByName,
		Metrics:     []baker.MetricsDesc{fooBarDesc},
	}
	cfg, err := baker.NewConfigFromToml(strings.NewReader(toml), c)
	if err != nil {
		log.Fatal(err)
	}

	if err := baker.Main(cfg); err != nil {
		log.Fatal(err)
	}
}
