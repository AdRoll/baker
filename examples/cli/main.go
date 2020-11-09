// CLI example illustrates how to build a Baker CLI program using baker.MainCLI
// Run without arguments to see the help. Run with options to explore them and
// run with the `example.toml` argument (the file in this folder) to execute the topology
package main

import (
	log "github.com/sirupsen/logrus"

	"github.com/AdRoll/baker"
	"github.com/AdRoll/baker/filter"
	"github.com/AdRoll/baker/input"
	"github.com/AdRoll/baker/output"
	"github.com/AdRoll/baker/upload"
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
	comp := baker.Components{
		Inputs:      input.All,
		Filters:     filter.All,
		Outputs:     output.All,
		Uploads:     upload.All,
		FieldByName: fieldByName,
	}

	if err := baker.MainCLI(comp); err != nil {
		log.Fatal(err)
	}
}
