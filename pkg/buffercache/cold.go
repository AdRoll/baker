package organizer

import (
	"encoding/binary"
	"math/bits"
	"sort"
)

// coldCache is a component of the BufferCache used to fastly store and
// retrieve log lines which grouping key has only been seen once.
//
// It is expected that a majority of grouping key are going to be unique,
// so the coldCache allows to store without any allocation the buffers for
// those unique keys.
// To minimize the waste of space the cold cache is divided into a series of
// buckets each of which can hold buffers of different sizes.
type coldCache struct {
	buckets []bucket
}

// newColdCache returns a coldCache with len(bsizes) buckets, each of which can
// hold ncells cells (or buffers), bsizes is the list of bucket cell sizes.
//
// NOTE: the following statements must be true of newColdCache returns an error:
// - the total number of cells must be a multiple of 64
// - bsizes must be sorted in ascending order
func newColdCache(ncells int, bsizes []int) (*coldCache, error) {
	if ncells == 0 || ncells%64 != 0 {
		return nil, errInvalidConfig("the number of cells per bucket must be positive and a multiple of 64")
	}

	// Ensure bucket sizes are sorted since, upon insertion of a buffer, we loop
	// them to find the first big enough to hold the buffer. It's also handy for
	// bucket metrics.
	if !sort.IntsAreSorted(bsizes) {
		return nil, errInvalidConfig("bucket sizes must be sorted in ascending order")
	}

	buckets := make([]bucket, 0)
	for _, bsize := range bsizes {
		buckets = append(buckets, newBucket(bsize, ncells))
	}
	return &coldCache{buckets: buckets}, nil
}

// smallestFitBucket returns the smallest bucket in which fits a buffer of length l
// or -1 if all buckets are too small.
func (c *coldCache) smallestFitBucket(l int) int {
	l = l + 4
	for i := range c.buckets {
		if c.buckets[i].cellbytes >= l {
			// This bucket is the first big enough to hold the buffer
			// prepended by the 4 bytes representing its length
			return i
		}
	}
	return -1
}

// metrics returns a snapshot of the cold cache metrics.
func (c *coldCache) metrics() *coldCacheMetrics {
	ratios := make([]float32, len(c.buckets))
	totbufs := uint64(0)
	totsize := uint64(0)
	for i := range c.buckets {
		nbufs := c.buckets[i].numBuffers()
		totbufs += uint64(nbufs)
		totsize += uint64(nbufs * c.buckets[i].cellbytes)
		ratios[i] = float32(nbufs) / float32(64*len(c.buckets[i].freecells))
	}

	return &coldCacheMetrics{
		TotalEntries: totbufs,
		TotalSize:    totsize,
		FillRatios:   ratios,
	}
}

// fillRatio returns the fill ratio of the bucket, that is the ratio of
// occupied cells over total cells.
func (b *bucket) fillRatio() float32 {
	ncells := 0
	for j := range b.freecells {
		ncells += bits.OnesCount64(b.freecells[j])
	}
	return float32(ncells) / float32(64*len(b.freecells))
}

type bucket struct {
	data      []byte // buffer, virtually split in cells
	cellbytes int    // number of bytes in a cell
	freecells bitmap // free cell bitmap (0:free, 1:occupied)
}

// newBucket instantiates a new bucket with ncells cells, each of which can
// hold 'cellbytes' bytes long (actually cellbytes - 4 since the buffer length
// if stored in the 4 first bytes).
func newBucket(cellbytes, ncells int) bucket {
	return bucket{
		cellbytes: cellbytes,
		data:      make([]byte, cellbytes*ncells),
		freecells: bitmap(make([]uint64, ncells/64)),
	}
}

// flush calls f with each buffer in this bucket, afterwards the bucket is
// considered free again.
func (b *bucket) flush(f func(buf []byte, compressed bool)) {
	for i := range b.freecells {
		for bit := 0; bit < 64; bit++ {
			if b.freecells[i]&(1<<bit) != 0 {
				buf, compressed := b.get((i * 64) + bit)
				f(buf, compressed)
			}
		}
	}

	// Clear the whole bitmap
	b.freecells = bitmap(make([]uint64, len(b.freecells)))
}

// put copies buf into b and returns the index of the cell where it has been
// copied or false if the bucket is currently full.
func (b *bucket) put(buf []byte, compressed bool) (int, bool) {
	idx, ok := b.freecells.findFreeCell()
	if !ok {
		return -1, false
	}
	pos := idx * b.cellbytes

	// Prefix the buffer with its size and the compressed bit.
	prefix := uint32(len(buf))
	if compressed {
		// set the compressed bit
		prefix |= 0x80000000
	}
	binary.LittleEndian.PutUint32(b.data[pos:], prefix)
	copy(b.data[pos+4:], buf)

	b.freecells.setBit(idx)
	return idx, true
}

// get returns the buffer held at a specific cell.
//
// NOTE: the returned slice directly points to the bucket backing array, so the
// slice content may be overwritten by later calls.
func (b *bucket) get(cellidx int) (buf []byte, compressed bool) {
	pos := uint32(cellidx * b.cellbytes)
	prefix := binary.LittleEndian.Uint32(b.data[pos : pos+4])
	buflen := prefix & 0x7fffffff
	compressed = (prefix & 0x80000000) != 0
	pos += 4
	return b.data[pos : pos+buflen], compressed
}

// numBuffers returns the number of buffers (i.e occupied cells) actually
// present in the bucket.
func (b *bucket) numBuffers() int {
	ncells := 0
	for j := range b.freecells {
		ncells += bits.OnesCount64(b.freecells[j])
	}
	return ncells
}
