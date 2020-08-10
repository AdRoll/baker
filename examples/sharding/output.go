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
func (s *Shardable) CanShard() bool {
	return true
}

func (s *Shardable) Run(input <-chan baker.OutputRecord, _ chan<- string) error {
	// Do something with `input` record.
	// s.idx identifies the output process index and should
	// be used to manage the sharding
	for data := range input {
		log.Printf(`Shard #%d: Getting "%s"`, s.idx, data.Record)
	}

	return nil
}

func (s *Shardable) Stats() baker.OutputStats { return baker.OutputStats{} }
