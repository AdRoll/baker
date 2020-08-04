// filtering example illustrates how to create a filter component.
// It takes the basic example adding a "Lazy" filter definition
package main

import (
	"log"
	"strings"
	"time"

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
[input]
name = "List"
[input.config]
	files=["./testdata/list-clause-files-comma-sep.input.csv.zst"]
[[filter]]
name = "LazyFilter"
	[filter.config]
	stakhanovite = true
[output]
name = "Files"
procs=1
    [output.config]
    PathString="./_out/list-clause-files-comma-sep.output.csv.gz"
	`
	c := baker.Components{
		Inputs:      input.All,
		Filters:     []baker.FilterDesc{LazyFilterDesc},
		Outputs:     output.All,
		FieldByName: fieldByName,
	}
	cfg, err := baker.NewConfigFromToml(strings.NewReader(toml), c)
	if err != nil {
		log.Fatal(err)
	}

	var duration time.Duration
	if err := baker.Main(cfg, duration); err != nil {
		log.Fatal(err)
	}
}
