package main

import (
	"hash/fnv"

	"github.com/AdRoll/baker"
)

func simpleHash(b []byte) uint64 {
	f := fnv.New64()
	f.Sum(b)
	return f.Sum64()
}

var shardingFuncs = map[baker.FieldIndex]baker.ShardingFunc{
	FieldA: fieldAToInt,
	FieldB: fieldBToInt,
	FieldC: fieldCToInt,
}

func fieldAToInt(r baker.Record) uint64 {
	f := r.Get(FieldA)
	return simpleHash(f)
}

func fieldBToInt(r baker.Record) uint64 {
	f := r.Get(FieldB)
	return simpleHash(f)
}

func fieldCToInt(r baker.Record) uint64 {
	f := r.Get(FieldC)
	return simpleHash(f)
}
