// advanced example shows an advanced setup of baker.Components
package main

import (
	"hash/fnv"
	"log"

	"github.com/AdRoll/baker"
	"github.com/AdRoll/baker/filter"
	"github.com/AdRoll/baker/input"
	"github.com/AdRoll/baker/output"
	"github.com/AdRoll/baker/upload"
)

func main() {
	if err := baker.MainCLI(components); err != nil {
		log.Fatal(err)
	}
}

// Some example fields
const (
	Timestamp baker.FieldIndex = 0
	Source    baker.FieldIndex = 1
	Target    baker.FieldIndex = 2
)

// And their respective names
var fieldNames = []string{"timestamp", "source", "target"}

var components = baker.Components{
	Inputs:        input.All,
	Filters:       filter.All,
	Outputs:       output.All,
	Uploads:       upload.All,
	ShardingFuncs: shardingFuncs,
	Validate:      validateLogLine,
	FieldByName:   fieldByName,
	FieldNames:    fieldNames,
}

var shardingFuncs = map[baker.FieldIndex]baker.ShardingFunc{
	Timestamp: timestampToInt,
	Source:    sourceToInt,
	Target:    targetToInt,
}

func timestampToInt(r baker.Record) uint64 {
	return simpleHash(r, Timestamp)
}

func sourceToInt(r baker.Record) uint64 {
	return simpleHash(r, Source)
}

func targetToInt(r baker.Record) uint64 {
	return simpleHash(r, Target)
}

func simpleHash(r baker.Record, idx baker.FieldIndex) uint64 {
	f := fnv.New64()
	f.Write(r.Get(idx))
	return f.Sum64()
}

func validateLogLine(baker.Record) (bool, baker.FieldIndex) {
	// All records are valid...
	return true, 0
}

func fieldByName(name string) (baker.FieldIndex, bool) {
	for idx, n := range fieldNames {
		if n == name {
			return baker.FieldIndex(idx), true
		}
	}

	return 0, false
}
