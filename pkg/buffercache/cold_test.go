package organizer

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestColdCacheGet(t *testing.T) {
	l0 := []byte("hello")
	l1 := []byte("world!!")
	bucket := newBucket(32, 64)
	i0, ok := bucket.put(l0, false)
	if !ok {
		t.Errorf("got full bucket but it shouldn't be")
	}
	i1, ok := bucket.put(l1, false)
	if !ok {
		t.Errorf("got full bucket but it shouldn't be")
	}
	buf, compressed := bucket.get(i0)
	if !bytes.Equal(buf, l0) {
		t.Errorf("got bucket.get(%d) = %s, want %s", i0, buf, l0)
	}
	if compressed {
		t.Errorf("got bucket.get(%d) compressed", i0)
	}

	buf, compressed = bucket.get(i1)
	if !bytes.Equal(buf, l1) {
		t.Errorf("got bucket.get(%d) = %s, want %s", i1, buf, l0)
	}
	if compressed {
		t.Errorf("got bucket.get(%d) compressed", i1)
	}
}

func TestColdCacheGet_BadCell(t *testing.T) {
	bucket := newBucket(32, 64)
	// trying to retrieve the buffer in an 'out-of-bound' cell should panic
	assertPanics(t, func() {
		bucket.get(64)
	})
}

func TestBufferCachePutDestination(t *testing.T) {
	tests := []struct {
		buflen             int
		wanthot            bool
		wantbidx, wantcidx int
	}{
		{
			buflen:   7,
			wantbidx: 1,
			wantcidx: 63,
		},
		{
			buflen:   0,
			wantbidx: 0,
			wantcidx: 63,
		},
		{
			buflen:   6,
			wantbidx: 0,
			wantcidx: 62,
		},
		{
			buflen:   9,
			wantbidx: 1,
			wantcidx: 62,
		},
		{
			buflen:   18,
			wantbidx: 2,
			wantcidx: 63,
		},
		{
			buflen:  28,
			wanthot: true,
		},
	}

	cache, _ := NewBufferCache(Config{
		CellsPerBucket:  64,
		Buckets:         []int{10, 20, 30},
		MaxBufferLength: 100,
		MaxCapacity:     200,
	})

	for tidx, tt := range tests {
		tname := fmt.Sprintf("%db buffer", tt.buflen)
		t.Run(tname, func(t *testing.T) {
			tbuf := incompressible[:tt.buflen]
			tkey := strconv.Itoa(tidx) // to have a different key for each test

			cache.Put(tkey, tbuf)
			if tt.wanthot {
				assertKeyInHotCache(t, cache, tkey)
				return
			}

			loc := cache.m.m[tkey]
			assertColdLocation(t, loc, tt.wantbidx, tt.wantcidx)
			assertKeyInColdCache(t, cache, tkey)
			assertKeyValue(t, cache, tkey, tbuf)
		})
	}
}

// genBuffer generates a compressible buffer, useful for testing,
// since lz4 library fails on incompressible data.
func genBuffer(s string, max int) []byte {
	if len(s) > max {
		panic(fmt.Sprintf("len(s) > max => len(%q) > %d", s, max))
	}
	return []byte(s + strings.Repeat("-", max-len(s)))
}

// 512 incompressible bytes for tests
var incompressible = []byte("\x52\xfd\xfc\x07\x21\x82\x65\x4f\x16\x3f\x5f\x0f\x9a\x62\x1d\x72\x95\x66\xc7\x4d\x10\x03\x7c\x4d\x7b\xbb\x04\x07\xd1\xe2\xc6\x49\x81\x85\x5a\xd8\x68\x1d\x0d\x86\xd1\xe9\x1e\x00\x16\x79\x39\xcb\x66\x94\xd2\xc4\x22\xac\xd2\x08\xa0\x07\x29\x39\x48\x7f\x69\x99\xeb\x9d\x18\xa4\x47\x84\x04\x5d\x87\xf3\xc6\x7c\xf2\x27\x46\xe9\x95\xaf\x5a\x25\x36\x79\x51\xba\xa2\xff\x6c\xd4\x71\xc4\x83\xf1\x5f\xb9\x0b\xad\xb3\x7c\x58\x21\xb6\xd9\x55\x26\xa4\x1a\x95\x04\x68\x0b\x4e\x7c\x8b\x76\x3a\x1b\x1d\x49\xd4\x95\x5c\x84\x86\x21\x63\x25\x25\x3f\xec\x73\x8d\xd7\xa9\xe2\x8b\xf9\x21\x11\x9c\x16\x0f\x07\x02\x44\x86\x15\xbb\xda\x08\x31\x3f\x6a\x8e\xb6\x68\xd2\x0b\xf5\x05\x98\x75\x92\x1e\x66\x8a\x5b\xdf\x2c\x7f\xc4\x84\x45\x92\xd2\x57\x2b\xcd\x06\x68\xd2\xd6\xc5\x2f\x50\x54\xe2\xd0\x83\x6b\xf8\x4c\x71\x74\xcb\x74\x76\x36\x4c\xc3\xdb\xd9\x68\xb0\xf7\x17\x2e\xd8\x57\x94\xbb\x35\x8b\x0c\x3b\x52\x5d\xa1\x78\x6f\x9f\xff\x09\x42\x79\xdb\x19\x44\xeb\xd7\xa1\x9d\x0f\x7b\xba\xcb\xe0\x25\x5a\xa5\xb7\xd4\x4b\xec\x40\xf8\x4c\x89\x2b\x9b\xff\xd4\x36\x29\xb0\x22\x3b\xee\xa5\xf4\xf7\x43\x91\xf4\x45\xd1\x5a\xfd\x42\x94\x04\x03\x74\xf6\x92\x4b\x98\xcb\xf8\x71\x3f\x8d\x96\x2d\x7c\x8d\x01\x91\x92\xc2\x42\x24\xe2\xca\xfc\xca\xe3\xa6\x1f\xb5\x86\xb1\x43\x23\xa6\xbc\x8f\x9e\x7d\xf1\xd9\x29\x33\x3f\xf9\x93\x93\x3b\xea\x6f\x5b\x3a\xf6\xde\x03\x74\x36\x6c\x47\x19\xe4\x3a\x1b\x06\x7d\x89\xbc\x7f\x01\xf1\xf5\x73\x98\x16\x59\xa4\x4f\xf1\x7a\x4c\x72\x15\xa3\xb5\x39\xeb\x1e\x58\x49\xc6\x07\x7d\xbb\x57\x22\xf5\x71\x7a\x28\x9a\x26\x6f\x97\x64\x79\x81\x99\x8e\xbe\xa8\x9c\x0b\x4b\x37\x39\x70\x11\x5e\x82\xed\x6f\x41\x25\xc8\xfa\x73\x11\xe4\xd7\xde\xfa\x92\x2d\xaa\xe7\x78\x66\x67\xf7\xe9\x36\xcd\x4f\x24\xab\xf7\xdf\x86\x6b\xaa\x56\x03\x83\x67\xad\x61\x45\xde\x1e\xe8\xf4\xa8\xb0\x99\x3e\xbd\xf8\x88\x3a\x0a\xd8\xbe\x9c\x39\x78\xb0\x48\x83\xe5\x6a\x15\x6a\x8d\xe5\x63\xaf\xa4\x67\xd4\x9d\xec\x6a\x40\xe9\xa1\xd0\x07\xf0\x33\xc2\x82\x30\x61\xbd\xd0\xea\xa5\x9f\x8e\x4d\xa6\x43\x01\x05\x22\x0d\x0b\x29\x68\x8b\x73\x4b\x8e\xa0\xf3\xca\x99\x36\xe8\x46\x1f\x10\xd7\x7c\x96\xea\x80\xa7\xa6")

func TestColdCacheFlushAuto(t *testing.T) {
	flushes := 0
	cfg := Config{
		CellsPerBucket:  64,
		Buckets:         []int{64},
		MaxBufferLength: 100,
		MaxCapacity:     200,
		OnFlush:         func([]byte) { flushes++ },
	}

	cache, _ := NewBufferCache(cfg)

	// Fill all cells of our bucket
	ikey := 0
	for ; ikey < cfg.CellsPerBucket; ikey++ {
		cache.Put(strconv.Itoa(ikey), genBuffer("ciao", 32))
	}
	if flushes != 0 {
		t.Errorf("got flushes = %d, want 0", flushes)
	}
	// Add another line, must flush now
	cache.Put(strconv.Itoa(ikey), genBuffer("ciao", 32))
	if flushes != 64 {
		t.Errorf("got flushes = %d, want: 64", flushes)
	}

	for i, b := range cache.cold.buckets {
		n := 0
		b.flush(func([]byte, bool) { n++ })
		if n > 1 {
			t.Errorf("got %d buffers in bucket %d, want 1", n, i)
		}
	}
}

func TestColdCacheSortedBuckets(t *testing.T) {
	t.Run("buckets-not-sorted", func(t *testing.T) {
		_, err := newColdCache(64, []int{2, 3, 1})
		if err == nil {
			t.Errorf("got err = nil, want error")
		}
	})
	t.Run("buckets-sorted", func(t *testing.T) {
		_, err := newColdCache(64, []int{2, 3, 4})
		if err != nil {
			t.Errorf("got err = %q, want nil", err)
		}
	})
}

func TestColdCacheMetrics(t *testing.T) {
	buf := make([]byte, 64)
	for i := range buf {
		buf[i] = byte(i)
	}

	c, _ := newColdCache(64, []int{16, 32, 64})

	// empty buckets (after init)
	metrics := c.metrics()
	want := &coldCacheMetrics{
		FillRatios:   []float32{0, 0, 0},
		TotalEntries: 0,
		TotalSize:    0,
	}
	if !reflect.DeepEqual(want, metrics) {
		t.Errorf("got metrics = %+v, want %+v", metrics, want)
	}

	// non-empty buckets
	c.buckets[0].put(buf[:1], false)
	c.buckets[0].put(buf[:1], false)
	c.buckets[0].put(buf[:1], false)
	c.buckets[1].put(buf[:17], false)
	c.buckets[1].put(buf[:17], false)
	c.buckets[2].put(buf[:33], false)

	metrics = c.metrics()
	want = &coldCacheMetrics{
		FillRatios:   []float32{3. / 64., 2. / 64., 1. / 64.},
		TotalEntries: 3 + 2 + 1,
		TotalSize:    3*16 + 2*32 + 64,
	}
	if !reflect.DeepEqual(want, metrics) {
		t.Errorf("got metrics = %+v, want %+v", metrics, want)
	}

	// full buckets
	for i := 0; i < 64; i++ {
		c.buckets[0].put(buf[:1], false)
		c.buckets[1].put(buf[:17], false)
		c.buckets[2].put(buf[:33], false)
	}

	metrics = c.metrics()
	want = &coldCacheMetrics{
		FillRatios:   []float32{1., 1., 1.},
		TotalEntries: 64 + 64 + 64,
		TotalSize:    64*16 + 64*32 + 64*64,
	}
	if !reflect.DeepEqual(want, metrics) {
		t.Errorf("got metrics = %+v, want %+v", metrics, want)
	}

	// empty buckets (after flush)
	c.buckets[0].flush(func([]byte, bool) {})
	c.buckets[1].flush(func([]byte, bool) {})
	c.buckets[2].flush(func([]byte, bool) {})

	metrics = c.metrics()
	want = &coldCacheMetrics{
		FillRatios: []float32{0, 0, 0},
	}
	if !reflect.DeepEqual(want, metrics) {
		t.Errorf("got metrics = %+v, want %+v", metrics, want)
	}
}
