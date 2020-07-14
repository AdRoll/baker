package main

import (
	"log"

	"github.com/AdRoll/baker"
)

var ShardableDesc = baker.OutputDesc{
	Name:   "Shardable",
	New:    NewShardable,
	Config: &ShardableConfig{},
	Raw:    true,
}

// A ShardableConfig specifies the Shardable configuration.
type ShardableConfig struct{}

// A Shardable output appends all received records, useful for examination in tests.
type Shardable struct {
	idx int
}

func NewShardable(cfg baker.OutputParams) (baker.Output, error) {
	return &Shardable{
		idx: cfg.Index,
	}, nil
}

// The output supports sharding
func (r *Shardable) CanShard() bool {
	return true
}

func (r *Shardable) Run(input <-chan baker.OutputRecord, _ chan<- string) {
	// Do something with the input record.
	// r.idx identifies the output process index and should
	// be used to manage the sharding
	for data := range input {
		log.Printf(`Shard #%d: Getting "%s"`, r.idx, data.Record)
	}
}

func (r *Shardable) Stats() baker.OutputStats { return baker.OutputStats{} }
