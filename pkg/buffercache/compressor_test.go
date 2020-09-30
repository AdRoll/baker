package buffercache

import (
	"bytes"
	"testing"
)

func Test_compressorCompressibleBuffer(t *testing.T) {
	c := compressor{}
	d := decompressor{}

	c.init()
	d.init()

	src := bytes.Repeat([]byte{'a', 'b', 'c'}, 32)
	buf, ok := c.compress(src)
	if !ok {
		t.Errorf("got ok=false, want true")
	}
	cpy := make([]byte, len(buf))
	copy(cpy, buf)

	got := d.uncompressSimple(cpy)
	if !bytes.Equal(src, got) {
		t.Errorf("got %q, want %q", got, src)
	}

	src = testBuffer
	buf, ok = c.compress(src)
	if !ok {
		t.Errorf("got ok=false, want true")
	}
	cpy = make([]byte, len(buf))
	copy(cpy, buf)

	got = d.uncompressSimple(cpy)
	if !bytes.Equal(src, got) {
		t.Errorf("got %q, want %q", got, src)
	}
}

func Test_decompressor_Copy(t *testing.T) {
	tests := []struct {
		pos  int
		b    string
		want string
	}{
		{
			pos:  0,
			b:    "hey",
			want: "hey\x00",
		},
		{
			pos:  1,
			b:    "hey",
			want: "\x00hey",
		},
		{
			pos:  2,
			b:    "heyYou",
			want: "\x00\x00heyYou",
		},
		{
			pos:  3,
			b:    "heyYou",
			want: "\x00\x00\x00heyYou",
		},
		{
			pos:  4,
			b:    "heyYou",
			want: "\x00\x00\x00\x00heyYou",
		},
		{
			pos:  5,
			b:    "hey",
			want: "\x00\x00\x00\x00\x00hey",
		},
	}

	for _, tt := range tests {
		d := decompressor{
			buf: make([]byte, 4),
		}

		d.copy(tt.pos, []byte(tt.b))

		if !bytes.Equal(d.buf, []byte(tt.want)) {
			t.Fatalf("pos=%d b=%q: got %v, want %v", tt.pos, tt.b, d.buf, []byte(tt.want))
		}
	}
}

func Test_decompressor_CopyByte(t *testing.T) {
	tests := []struct {
		pos  int
		c    byte
		want string
	}{
		{
			pos:  0,
			c:    '*',
			want: "*\x00\x00\x00",
		},
		{
			pos:  1,
			c:    '*',
			want: "\x00*\x00\x00",
		},
		{
			pos:  2,
			c:    '*',
			want: "\x00\x00*\x00",
		},
		{
			pos:  3,
			c:    '*',
			want: "\x00\x00\x00*",
		},
		{
			pos:  4,
			c:    '*',
			want: "\x00\x00\x00\x00*",
		},
		{
			pos:  6,
			c:    '*',
			want: "\x00\x00\x00\x00\x00\x00*",
		},
	}

	for _, tt := range tests {
		d := decompressor{
			buf: make([]byte, 4),
		}

		d.copyByte(tt.pos, tt.c)

		if !bytes.Equal(d.buf, []byte(tt.want)) {
			t.Fatalf("pos=%d b=%c: got %v, want %v", tt.pos, tt.c, d.buf, []byte(tt.want))
		}
	}
}

func Test_compressorBufferGrow(t *testing.T) {
	// Start with a very small internal buffer to ensure we grow it
	c := compressor{
		ht:  make([]int, lz4TableSize),
		buf: make([]byte, 4),
	}

	buf := bytes.Repeat([]byte("dummyData"), 100)
	zbuf, ok := c.compress(buf)
	if !ok {
		t.Errorf("c.compress(buf): got ok=false want true")
	}
	if zbuf == nil {
		t.Errorf("c.compress(buf): got zbuf=nil want compressed data")
	}

	// Same for decompressor: use a very small internal buffer
	d := decompressor{
		buf: make([]byte, 4),
	}

	got := d.uncompressSimple(zbuf)
	if !bytes.Equal(got, buf) {
		t.Errorf("uncompress: got %q want %q", got, buf)
	}
}

func Test_compressorIncompressibleData(t *testing.T) {
	// Volontarily make the compressor internal buffer too small to hold the
	// compressed data so we trigger the 'compress' return values indicating
	// incompressible data.
	c := compressor{}
	c.buf = make([]byte, 1)
	c.ht = make([]int, lz4TableSize)

	zbuf, ok := c.compress([]byte{'a'})
	if ok {
		t.Errorf("c.compress('a''): got ok=true want false (incompressible data)")
	}
	if zbuf != nil {
		t.Errorf("c.compress(buf): got zbuf=%q want nil (incompressible data)", zbuf)
	}
}

var testBuffer []byte = []byte(`foo0123456789dhtbf1267485145JFLGHTB123456defea012345678901234567890101234567890123456789015555555f0123456789012345678901foo-bar-bazcrm[]`)
var testBufferZ []byte

func init() {
	c := compressor{}
	c.init()

	var ok bool
	testBufferZ, ok = c.compress(testBuffer)
	if !ok {
		panic("couldn't compressed testBufferZ")
	}
}

func Benchmark_compressor(b *testing.B) {
	c := compressor{}
	c.init()

	for n := 0; n < b.N; n++ {
		c.compress(testBuffer)
	}
}

func Benchmark_decompressor(b *testing.B) {
	d := decompressor{}
	d.init()

	for n := 0; n < b.N; n++ {
		d.uncompressSimple(testBufferZ)
	}
}
