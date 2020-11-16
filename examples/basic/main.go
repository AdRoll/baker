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
[input]
name = "List"

	[input.config]
	files=["testdata/input.csv.zst"]

[fields]
names=["timestamp", "source", "target"]

[[filter]]
name="ReplaceFields"
	[filter.config]
	ReplaceFields=["replaced", "timestamp"]

[output]
name = "FileWriter"
procs=1

	[output.config]
	PathString="./_out/output.csv.gz"
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
