package main

import (
	"log"
	"strings"
	"time"

	"github.com/AdRoll/baker"
	"github.com/AdRoll/baker/input"
	"github.com/AdRoll/baker/output"
)

var fields = map[string]baker.FieldIndex{
	"fieldA": 0,
	"fieldB": 1,
	"fieldC": 2,
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
    files=["./examples/data/list-clause-files-comma-sep.csv.zst"]
[output]
name = "Files"
procs=1
    [output.config]
    PathString="./_out/output-list-clause-files-comma-sep.csv.gz"
	`
	c := baker.Components{
		Inputs:      input.All,
		Outputs:     output.All,
		FieldByName: fieldByName,
	}
	cfg, err := baker.NewConfigFromToml(strings.NewReader(toml), c)
	if err != nil {
		log.Fatal(err)
	}
	var duration time.Duration
	err = baker.Main(cfg, duration)
	if err != nil {
		log.Fatal(err)
	}
}
