package buffercache

import "fmt"

// A location represents the location of an entry in the cache.
//
// It can either point to an entry in the hot or in the cold cache.
// The MSB indicates whether the entry pointed to by this location is
// in the hot cache.
// If the location indicates a cold cache entry, then the bucket index
// is at the bits 24-30, while the 24 LSB represents the cell index
// in that bucket.
//
//  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//  |H| BucketIndex |                   CellIndex                   |
//  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//   1|   7 bits    |                     24 bits                   |
type location uint32

// hot cache sentinel value
const hotCacheLocation location = 1 << 31

// isHot/isCold reports whether l points to an entry in the hot/cold cache.
func (l location) isCold() bool { return l&0x80000000 == 0 }
func (l location) isHot() bool  { return l&0x80000000 != 0 }

// coldBucketIdx/coldCellIdx returns the bucket/cell index in cold cache.
func (l location) coldBucketIdx() int { return int(l >> 24) }
func (l location) coldCellIdx() int   { return int(l & 0x00ffffff) }

func (l location) String() string {
	return fmt.Sprintf("0x%x", uint32(l))
}

// coldLocation returns a location pointing to a cold cache entry.
func coldLocation(bidx, cidx int) location {
	return location(bidx)<<24 | location(cidx)
}

// a locationMap maps keys to locations.
type locationMap struct {
	m map[string]location
}

func (m *locationMap) reset() {
	m.m = make(map[string]location)
}

// removeCold removes all locations pointing to the given cold cache bucket.
func (m *locationMap) removeCold(bidx int) {
	for key, loc := range m.m {
		if loc.isHot() {
			continue
		}

		if bidx == loc.coldBucketIdx() {
			delete(m.m, key)
		}
	}
}

// removeHot removes all locations pointing to the hot cache.
func (m *locationMap) removeHot() {
	for key, loc := range m.m {
		if loc.isHot() {
			delete(m.m, key)
		}
	}
}
