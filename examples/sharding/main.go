// sharding example shows how to use an output that supports sharding
// When a shardable output is used, the parallel outputs
// identified by the "procs" configuration value in the toml,
// receive a subset of the processed records. The sharding function
// returns a shard idx (based on the sharded field value) which value
// is assigned to an output process calculating the modulo with the
// available output processes.
// This means that using a procs=1 configuration is the same as disabling
// the sharding, while procs=N where N is the number of possible values
// of the sharded field means that each output receives records with always
// the same value for that field
package main

import (
	"log"
	"strings"

	"github.com/AdRoll/baker"
	"github.com/AdRoll/baker/input"
)

// Some example fields
const (
	ID        baker.FieldIndex = 0
	FirstName baker.FieldIndex = 1
	LastName  baker.FieldIndex = 2
	Age       baker.FieldIndex = 3
	Street    baker.FieldIndex = 4
	City      baker.FieldIndex = 5
	Dollar    baker.FieldIndex = 6
)

// And their respective names
var fieldNames = []string{
	"id",
	"first_name",
	"last_name",
	"age",
	"street",
	"city",
	"dollar",
}

func fieldByName(name string) (baker.FieldIndex, bool) {
	for idx, fname := range fieldNames {
		if name == fname {
			return baker.FieldIndex(idx), true
		}
	}

	return 0, false
}

var components = baker.Components{
	Inputs:        input.All,
	Outputs:       []baker.OutputDesc{ShardableDesc},
	ShardingFuncs: shardingFuncs,
	FieldByName:   fieldByName,
	FieldNames:    fieldNames,
}

func main() {
	toml := `
[input]
name="List"

	[input.config]
	files=["./testdata/customers_random.input.csv.zst"]

[output]
name="Shardable"
sharding="age" # "city" can be used as well
procs=10
`
	cfg, err := baker.NewConfigFromToml(strings.NewReader(toml), components)
	if err != nil {
		log.Fatal(err)
	}

	if err := baker.Main(cfg); err != nil {
		log.Fatal(err)
	}
}
