// Package buffercache provides the BufferCache type, a kind of map[string][]byte
// that is optimized for appending new bytes to the cache values. BufferCache
// has a maximum capacity and flushes itself when it's reached and calling a user-
// provided callback with the flushed buffers.
//
// BufferCache also supports compressing the byte buffers it holds with lz4 in
// order to trade cpu for memory, buffers are transparently uncompressed
// when the flush callback is called.
package buffercache

import (
	"encoding/binary"
	"fmt"
)

// Config configures a BufferCache.
type Config struct {

	// MaxCapacity is the maximum size in bytes the BufferCache can hold
	// without in dynamic memory (hot cache). This does not take into account
	// the buckets memory.
	MaxCapacity int

	// MaxBufferLength is the maximum size the buffer of a single key can reach
	// without being flushed.
	MaxBufferLength int

	// CellsPerBucket is the number of cells (or buffers) a bucket can contain.
	// It is expected that a majority of keys are going to hold a single buffer,
	// so each of those buffers can directly copied in pre-allocated memory
	// regions, called cells, which are limited in size. A bucket group cells
	// of the same size together.
	//
	// NOTE: this number must be a multiple of 64 or the configuration is invalid.
	CellsPerBucket int

	// Buckets indicates the cell sizes in bytes, for the various buckets.
	Buckets []int

	// OnFlush is the function called to flush the cache content when more
	// memory is required. It is called once per key with the whole buffer for
	// that key.
	OnFlush func([]byte)

	// EnableCompression indicates whether buffers should be compressed in memory
	// with a fast compression algorithm. This option is transparent, meaning
	// buffers are uncompressed in their original form when flushed. Hence
	// enabling this option trades CPU for memory.
	EnableCompression bool
}

// A BufferCache is like a map[string][]byte that is optimized for appending
// new bytes to the cache values.
type BufferCache struct {
	m    locationMap
	cold *coldCache
	hot  struct {
		m           map[string][]byte // actual cache
		cap, maxcap int               // current and total capacity (sum of all buffers)
		maxbuflen   int               // max length allowed for a single buffer
	}

	// onFlush is called each time a buffer is evicted from the cache.
	// nflushes is the number of times onFlush is called.
	onFlush  func([]byte)
	nflushes uint64

	compressionOn bool
	comp          compressor
	decomp        decompressor
}

type errInvalidConfig string

func (e errInvalidConfig) Error() string {
	return "BufferCache: invalid config: " + string(e)
}

// New creates a new BufferCache.
//
// A BufferCache is made of 2 components, a cold cache for keys that have been
// presented to the cache only once, and a hot cache for the keys having been
// seen more than once. In other words, when putting a (key, buffer) pair into
// the cache, the buffer is either copied into the cold cache if the key is not
// present already, or, in case a buffer already exists within the cache for that
// key, the new buffer gets appended to the previous one, separated by '\n'.
//
// The cold cache is flushed when it's full.
//
// The hot cache is flushed on the following occasions:
// - upon appending a buffer, the total buffer for that key would exceed
// maxBufferLength; in that case the previous buffer is passed to flush.
// - upon the insertion of new (key, buffer) pair, the total capacity of the
// cache would exceed maxCapacity; in that case the whole cache is empty (i.e
// buffers of all keys are passed to flush).
func New(cfg Config) (*BufferCache, error) {
	if cfg.MaxBufferLength < 0 {
		return nil, errInvalidConfig("maxBufferLength must be positive")
	}
	if cfg.MaxCapacity < 0 {
		return nil, errInvalidConfig("MaxCapacity must be positive")
	}
	if cfg.MaxBufferLength > cfg.MaxCapacity {
		return nil, errInvalidConfig("maxBufferLength can't exceed MaxCapacity")
	}

	hot := struct {
		m           map[string][]byte
		cap, maxcap int
		maxbuflen   int
	}{
		m:         make(map[string][]byte),
		maxbuflen: cfg.MaxBufferLength,
		maxcap:    cfg.MaxCapacity,
	}

	cold, err := newColdCache(cfg.CellsPerBucket, cfg.Buckets)
	if err != nil {
		return nil, err
	}

	if cfg.OnFlush == nil {
		cfg.OnFlush = func([]byte) {}
	}

	c := &BufferCache{
		m:             locationMap{m: make(map[string]location)},
		cold:          cold,
		hot:           hot,
		compressionOn: cfg.EnableCompression,
	}

	// configure lz4 compressor/decompressor
	c.comp.init()
	c.decomp.init()

	c.onFlush = func(buf []byte) {
		cfg.OnFlush(buf)
		c.nflushes++
	}
	return c, nil
}

// Put puts buf in the cache.
//
// Either a new index, key, is created with buf, or buf is appended to the
// previous buffer that is indexed by key.
//
// This may trigger flush(es).
func (c *BufferCache) Put(key string, buf []byte) {
	compressed := false
	if c.compressionOn {
		var zbuf []byte
		zbuf, compressed = c.comp.compress(buf)
		if compressed {
			buf = zbuf
		}
	}

	// Is it in the cache already?
	loc, ok := c.m.m[key]
	if ok {
		// Find it
		if loc.isCold() {
			c.moveToHotCache(key, loc)
		}
		c.putInHot(key, buf, compressed)
		return
	}
	c.putInCold(key, buf, compressed)
}

// Flush flushes the whole cache.
func (c *BufferCache) Flush() {
	c.flushCold()
	c.flushHot()
	c.m.reset()
}

func (c *BufferCache) flushHot() {
	for _, buf := range c.hot.m {
		c.flushBlocks(buf)
	}
	c.hot.m = make(map[string][]byte)
	c.hot.cap = 0
	c.m.removeHot()
}

func (c *BufferCache) decompressAndFlush(buf []byte, compressed bool) {
	if !compressed {
		c.onFlush(buf)
		return
	}

	c.onFlush(c.decomp.uncompressSimple(buf))
}

// flushBlocks decompresses and flushes consecutive blocks in buf, interleaving each of
// them with a `\n` in the decompressed buffer.
func (c *BufferCache) flushBlocks(buf []byte) {
	var (
		pos  uint64
		zpos int // indices on buf and zbuf
		n    int // length of last decompressed buffer
	)

	// Decompress consecutive blocks. Each block is prepended with an uint32
	// representing the block length. The outer loop moves pos and zpos.
	// The inner loop handles failed decompression due to not enough space in
	// the destination buffer to hold the currently decompressed block.
	for pos != uint64(len(buf)) {
		// Extract block length from the first 4 bytes.
		prefix := uint64(binary.LittleEndian.Uint32(buf[pos : pos+4]))
		blocklen, compressed := prefix&0x7fffffff, (prefix&0x80000000) != 0
		pos += 4

		if !compressed { // just copy it
			for {
				if uint64(len(c.decomp.buf[zpos:])) < blocklen {
					c.decomp.grow()
					continue
				}
				n = copy(c.decomp.buf[zpos:], buf[pos:pos+blocklen])
				break
			}
		} else {
			n = c.decomp.uncompress(zpos, buf[pos:pos+blocklen])
		}

		pos += blocklen
		zpos += n

		c.decomp.copyByte(zpos, '\n')
		zpos++
	}

	// Return the effective part of c.zbuf
	c.onFlush(c.decomp.bytes(zpos))
}

func (c *BufferCache) flushCold() {
	for b := range c.cold.buckets {
		c.cold.buckets[b].flush(c.decompressAndFlush)
	}
}

// putInCold puts buf in the cold cache if its size fits it, otherwise the
// buffer is put into the hot cache.
func (c *BufferCache) putInCold(key string, buf []byte, compressed bool) {
	bidx := c.cold.smallestFitBucket(len(buf))
	if bidx < 0 {
		// can't fit in no bucket, only destination is the hot cache.
		c.m.m[key] = hotCacheLocation
		c.putInHot(key, buf, compressed)
		return
	}

	// First try: find a free cell and copy buf into it
	cidx, ok := c.cold.buckets[bidx].put(buf, compressed)
	if !ok {
		// Bucket is full: flush it
		c.cold.buckets[bidx].flush(c.decompressAndFlush)
		c.m.removeCold(bidx)
		cidx, ok = c.cold.buckets[bidx].put(buf, compressed)
		if !ok {
			panic("still can't find a free cell in a bucket we just flushed")
		}
	}
	// Update the location entry
	c.m.m[key] = coldLocation(bidx, cidx)
}

// putInHot places buf in the hot cache, copying it into a new hot cache buffer
// if that's the first with that key, or appending to the previous buffer with
// that key, with a separator in between them..
func (c *BufferCache) putInHot(key string, buf []byte, compressed bool) {
	blen := len(buf)
	if blen+4 > c.hot.maxbuflen {
		panic(fmt.Sprintf("BufferCache: can't add buffers bigger than %d bytes", c.hot.maxbuflen))
	}

	if c.hot.cap+blen+4 > c.hot.maxcap {
		// Adding this would exceed maximum hot cache capacity so we flush
		// the whole cache.
		c.flushHot()
		// After flushing, this key would now be unique in the whole cache so
		// it goes by definition in the cold cache.
		c.putInCold(key, buf, compressed)
		return
	}

	bbuf, ok := c.hot.m[key]
	if !ok {
		// Create a hot cache entry
		bbuf := make([]byte, blen+4)

		// Prefix the buffer with its size and the compressed bit.
		prefix := uint32(blen)
		if compressed {
			// set the compressed bit
			prefix |= 0x80000000
		}

		binary.LittleEndian.PutUint32(bbuf[:4], prefix)
		copy(bbuf[4:], buf)
		c.hot.m[key] = bbuf
		c.hot.cap += blen + 4
		return
	}

	bblen := len(bbuf)
	if blen+bblen+4 > c.hot.maxbuflen {
		// Adding this would exceed maximum hot buffer length so we flush this
		// entry, removing both its location from the location map and the
		// actual buffer from the hot cache.
		delete(c.m.m, key)
		delete(c.hot.m, key)
		c.flushBlocks(bbuf)
		c.hot.cap -= bblen

		// After flushing, the new key would now be unique in the whole cache,
		// so by definition, its destination is the cold cache.
		c.putInCold(key, buf, compressed)
		return
	}

	// Grow current entry before appending to it:
	// - 4 bytes representing the prefix of the new buffer (length + compressed bit)
	// - the new buffer itself
	bbuf = append(bbuf, make([]byte, 4+blen)...)
	prefix := uint32(blen)
	if compressed {
		// set the compressed bit
		prefix |= 0x80000000
	}
	binary.LittleEndian.PutUint32(bbuf[bblen:bblen+4], prefix)
	copy(bbuf[bblen+4:], buf)
	c.hot.m[key] = bbuf
	c.hot.cap += blen + 4
}

// moveToHotCache moves a buffer from the cold to the hot cache.
// loc represents the location of the buffer in the cold cache.
func (c *BufferCache) moveToHotCache(key string, loc location) {
	// Extract buffer and cell index from the location
	bidx, cidx := loc.coldBucketIdx(), loc.coldCellIdx()

	// Retrieve the log line
	l, compressed := c.cold.buckets[bidx].get(cidx)
	// Clear the corresponding cell in the bitmap
	c.cold.buckets[bidx].freecells.clearBit(cidx)
	c.putInHot(key, l, compressed)
	c.m.m[key] = hotCacheLocation
}

// Metrics returns a snapshot of the cache performance counters.
//
// This is not safe for concurrent use by multiple goroutines.
func (c *BufferCache) Metrics() Metrics {
	m := Metrics{}

	m.Hot.TotalEntries = uint64(len(c.hot.m))
	m.Hot.TotalSize = uint64(c.hot.cap)
	m.Cold = *c.cold.metrics()
	m.TotalFlushes = c.nflushes
	return m
}
