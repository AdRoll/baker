package organizer

import (
	"fmt"

	"github.com/pierrec/lz4/v3"
)

const lz4TableSize = 64 << 10
const compressedBufferSize = 1024 * 1024 // 1MB

// a compressor is an lz4 compressor that is optimized for reducing memory
// allocations.
type compressor struct {
	ht  []int  // hash table
	buf []byte // re-used as destination when compressing
}

func (c *compressor) init() {
	c.ht = make([]int, lz4TableSize)
	c.buf = make([]byte, compressedBufferSize)
}

// compress compresses b and returns the compressed buffer. The compressed
// buffer is only valid until the next call to compres.
//
// If b is not-compressible (i.e the compressed buffer would actually be
// bigger than the source) compress returns (nil, false).
func (c *compressor) compress(buf []byte) (zbuf []byte, ok bool) {
	c.buf = c.buf[:cap(c.buf)]

	// Try to compress block
	n, err := lz4.CompressBlock(buf, c.buf, c.ht)
	if n == 0 && err == nil {
		return nil, false // buf is not compressible
	}

	// For some reason we couldn't buf, we're going to retry with a larger buffer.
	if err == lz4.ErrInvalidSourceShortBuffer {
		// compressedBufferSize should be dimensioned so this should nearly
		// never happen, but in case a very large buffer made its way here,
		// we're going to need a larger buffer for compression. So we grow the
		// internal buffer until compression success or we're sure the buffer
		// is big enough.
		maxbound := lz4.CompressBlockBound(len(buf))
		for len(c.buf) < maxbound {
			n, err = lz4.CompressBlock(buf, c.buf, c.ht)
			if err == lz4.ErrInvalidSourceShortBuffer {
				c.grow()
				continue
			}
			if err != nil {
				// We don't know how to recover from this.
				panic(fmt.Sprintf("lz4 compress on src block of length %d: %s", len(buf), err))
			}
			break
		}
	}
	return c.buf[:n], true
}

func (c *compressor) grow() {
	// Increase internal buffer capacity by 50%
	c.buf = append(c.buf, make([]byte, cap(c.buf)/2)...)
	c.buf = c.buf[:cap(c.buf)]
}

const uncompressedBufferSize = compressedBufferSize * 10

// a decompressor is an lz4 decompressor that is optimized for reducing memory
// allocations.
type decompressor struct {
	buf []byte // re-used as destination when uncompressing
}

func (d *decompressor) init() {
	d.buf = make([]byte, compressedBufferSize)
}

// uncompressSimple decompresses b and returns the resulting buffer.
// The returned buffer is only valid until the next call to uncompress.
func (d *decompressor) uncompressSimple(buf []byte) []byte {
	n := d.uncompress(0, buf)
	return d.buf[:n]
}

// uncompress uncompresses b into the decompressor internal buffer,
// starting at pos. Returns how many uncompressed bytes have been written.
func (d *decompressor) uncompress(pos int, buf []byte) int {
	for {
		n, err := lz4.UncompressBlock(buf, d.buf[pos:])
		if err == nil {
			return n
		}
		if err == lz4.ErrInvalidSourceShortBuffer {
			d.grow()
			continue
		}

		panic(fmt.Sprintf("uncompress: n:%d err:%s", n, err))
	}
}

func (d *decompressor) grow() {
	// Increase internal buffer capacity by 50%
	d.buf = append(d.buf, make([]byte, cap(d.buf)/2)...)
	d.buf = d.buf[:cap(d.buf)]
}

// bytes returns the decompressor internal buffer, resliced up until byte n.
func (d *decompressor) bytes(n int) []byte {
	return d.buf[:n]
}

// copy copies buf at index pos of the internal buffer, growing it if necessary.
func (d *decompressor) copy(pos int, buf []byte) {
	blen := len(buf)
	dblen := len(d.buf)
	if pos+blen > dblen {
		d.buf = append(d.buf, make([]byte, blen+pos-dblen)...)
	}
	copy(d.buf[pos:], buf)
}

// copyByte copies c at index pos of the internal buffer, growing it if necessary.
func (d *decompressor) copyByte(pos int, c byte) {
	dblen := len(d.buf)
	if pos+1 > dblen {
		d.buf = append(d.buf, make([]byte, 1+pos-dblen)...)
	}
	d.buf[pos] = c
}
