package buffercache

// Metrics holds a snapshot of the performance counters of a BufferCache instance.
type Metrics struct {
	Cold         coldCacheMetrics
	Hot          hotCacheMetrics
	TotalFlushes uint64 // flushes since cache creation
}

type coldCacheMetrics struct {
	TotalEntries uint64    // total number of entries (buffers) stored in cold cache
	TotalSize    uint64    // size of all occupied cells (doesn't consider free space inside of cells)
	FillRatios   []float32 // per bucket fill ratio
}

type hotCacheMetrics struct {
	TotalEntries uint64 // total number of entries (buffers) in hot cache
	TotalSize    uint64 // cummulated size of all buffers
}
