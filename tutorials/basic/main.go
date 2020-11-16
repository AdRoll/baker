package main

import (
	"log"

	"github.com/AdRoll/baker"
	"github.com/AdRoll/baker/filter"
	"github.com/AdRoll/baker/input"
	"github.com/AdRoll/baker/output"
	"github.com/AdRoll/baker/upload"
)

func main() {
	components := baker.Components{
		Inputs:      input.All,
		Filters:     filter.All,
		Outputs:     output.All,
		Uploads:     upload.All,
		FieldByName: fieldByName,
		FieldName:   fieldName,
	}
	if err := baker.MainCLI(components); err != nil {
		log.Fatal(err)
	}
}

var fields = []string{
	"review_num",
	"brand",
	"variety",
	"style",
	"country",
	"stars",
	"top_ten",
}

var fieldsByName = map[string]baker.FieldIndex{
	"review_num": 0,
	"brand":      1,
	"variety":    2,
	"style":      3,
	"country":    4,
	"stars":      5,
	"top_ten":    6,
}

func fieldByName(name string) (idx baker.FieldIndex, ok bool) {
	idx, ok = fieldsByName[name]
	return idx, ok
}

func fieldName(idx baker.FieldIndex) string {
	return fields[idx]
}
