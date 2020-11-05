// advanced example shows a complex setup of baker.Components
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

func simpleHash(r baker.Record, idx baker.FieldIndex) uint64 {
	f := fnv.New64()
	f.Write(r.Get(idx))
	return f.Sum64()
}

// Some example fields
const (
	Timestamp baker.FieldIndex = 0
	Source    baker.FieldIndex = 1
	Target    baker.FieldIndex = 2
)

var components = baker.Components{
	Inputs:        input.All,
	Filters:       filter.All,
	Outputs:       output.All,
	Uploads:       upload.All,
	ShardingFuncs: shardingFuncs,
	Validate:      validateLogLine,
	FieldByName:   fieldByName,
	FieldName:     fieldName,
}

var fields = map[string]baker.FieldIndex{
	"timestamp": Timestamp,
	"source":    Source,
	"target":    Target,
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

func validateLogLine(baker.Record) (bool, baker.FieldIndex) {
	// All records are valid...
	return true, 0
}

func fieldByName(key string) (baker.FieldIndex, bool) {
	idx, ok := fields[key]
	return idx, ok
}

func fieldName(idx baker.FieldIndex) string {
	for k, v := range fields {
		if v == idx {
			return k
		}
	}
	return ""
}
