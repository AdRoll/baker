package main

import (
	"github.com/AdRoll/baker"
)

var hextable [256]uint8

func init() {
	for i := range hextable {
		hextable[i] = 0xFF
	}
	for i := 'a'; i <= 'f'; i++ {
		hextable[i] = uint8(i) - 'a' + 10
	}
	for i := 'A'; i <= 'F'; i++ {
		hextable[i] = uint8(i) - 'A' + 10
	}
	for i := '0'; i <= '9'; i++ {
		hextable[i] = uint8(i) - '0'
	}
}

func uuidToInt(uuid []byte) uint64 {
	n := len(uuid)
	hash := uint64(5381)

	for i := 0; i < n; i++ {
		v := hextable[uuid[i]]
		v *= 16
		i++

		c := hextable[uuid[i]]
		v += c
		hash = hash*33 + uint64(v)
	}

	return hash
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
