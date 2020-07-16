package main

import (
	"hash/fnv"

	"github.com/AdRoll/baker"
)

// We support sharding by Age or City
var shardingFuncs = map[baker.FieldIndex]baker.ShardingFunc{
	Age:  ageToInt,
	City: cityToInt,
}

func ageToInt(r baker.Record) uint64 {
	return simpleHash(r, Age)
}

func cityToInt(r baker.Record) uint64 {
	return simpleHash(r, City)
}

func simpleHash(r baker.Record, idx baker.FieldIndex) uint64 {
	f := fnv.New64()
	f.Write(r.Get(idx))
	return f.Sum64()
}
