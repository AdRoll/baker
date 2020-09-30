package buffercache

import (
	"math/bits"
)

// a bitmap is a bit array where bits correspond to cells, 0 indicates a free
// cell and 1 t a busy one.
type bitmap []uint64

const numBits = 64

// findFreeCell returns the index of the first free cell or false.
func (bm *bitmap) findFreeCell() (int, bool) {
	// false if no cell is free.
	for i, slice := range *bm {
		index, ok := findFirstZero64(slice)
		if ok {
			return i*numBits + index, true
		}
	}
	return 0, false
}

// findFirstZero64 returns the index of the first bit at 0, starting from the
// LSB (least significant bit).
func findFirstZero64(x uint64) (int, bool) {
	l := bits.LeadingZeros64(x)
	if l != 0 {
		return 63, true
	}
	l = bits.LeadingZeros64(^x)
	if l == 64 {
		return 0, false
	}
	return int(63 - l), true
}

// setBit sets bit i (0-based, starting from LSB)
func (bm *bitmap) setBit(i int) {
	index := i / 64
	bit := i % 64

	(*bm)[index] |= 1 << bit
}

// clearBit clears bit i (0-based, starting from LSB)
func (bm *bitmap) clearBit(i int) {
	index := i / 64
	bit := i % 64

	(*bm)[index] &= ^(1 << bit)
}
