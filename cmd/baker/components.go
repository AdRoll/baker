package main

import (
	"github.com/AdRoll/baker"
	"github.com/AdRoll/baker/filter"
	"github.com/AdRoll/baker/input"
	"github.com/AdRoll/baker/output"
)

const (
	FieldA baker.FieldIndex = 0
	FieldB baker.FieldIndex = 1
	FieldC baker.FieldIndex = 2
)

var components = baker.Components{
	Inputs:        input.All,
	Filters:       filter.All,
	Outputs:       output.All,
	ShardingFuncs: shardingFuncs,
	Validate:      validateLogLine,
	FieldByName:   fieldByName,
	FieldName:     fieldName,
}

var fields = map[string]baker.FieldIndex{
	"fieldA": FieldA,
	"fieldB": FieldB,
	"fieldC": FieldC,
}

var shardingFuncs = map[baker.FieldIndex]baker.ShardingFunc{
	FieldA: fieldAToInt,
	FieldB: fieldBToInt,
	FieldC: fieldCToInt,
}

func fieldAToInt(r baker.Record) uint64 {
	f := r.Get(FieldA)
	return uuidToInt(f)
}

func fieldBToInt(r baker.Record) uint64 {
	f := r.Get(FieldB)
	return uuidToInt(f)
}

func fieldCToInt(r baker.Record) uint64 {
	f := r.Get(FieldC)
	return uuidToInt(f)
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
