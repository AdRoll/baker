package buffercache

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"
)

func Test_NewCache(t *testing.T) {
	cfg := Config{
		MaxBufferLength: 128,
		MaxCapacity:     256,
		CellsPerBucket:  64,
		OnFlush:         func([]byte) {},
	}

	c, _ := New(cfg)
	if len(c.m.m) != 0 {
		t.Errorf("Wrong hm: %v", c.m.m)
	}
	if c.cold == nil {
		t.Errorf("Wrong cold cache initialization")
	}
	if c.hot.maxbuflen != cfg.MaxBufferLength {
		t.Errorf("got c.maxBufferLength = %d, want: %d", c.hot.maxbuflen, cfg.MaxBufferLength)
	}
	if c.hot.maxcap != cfg.MaxCapacity {
		t.Errorf("got c.maxCapacity = %d, want: %d", c.hot.maxcap, cfg.MaxCapacity)
	}
}

func Test_isInColdCache(t *testing.T) {
	tests := []struct {
		name string
		loc  location
		want bool
	}{
		{"all zeroes", 0b00000000000000000000000000000000, true},
		{"leading one", 0b10000000000000000000000000000000, false},
		{"all ones", 0b11111111111111111111111111111111, false},
		{"all ones but first", 0b01111111111111111111111111111111, true},
		{"all zeros with less specified bits", 0b0, true},
		{"trailing one", 0b1, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.loc.isCold() != tt.want {
				t.Errorf("got loc.isCold() = %t, want %t", tt.loc.isCold(), tt.want)
			}
		})
	}
}

func TestBufferCache_PutAndFlushCompressed(t *testing.T) {
	flushed := false
	cfg := Config{
		MaxBufferLength:   128,
		MaxCapacity:       512,
		CellsPerBucket:    64,
		Buckets:           []int{64},
		OnFlush:           func([]byte) { flushed = true },
		EnableCompression: true,
	}

	c, _ := New(cfg)

	// Put a line in the cache for key1, must be in the cold cache
	c.Put("key1", genBuffer("ciao", 32))
	if c.cold.buckets[0].freecells.bit(63) != 1 {
		t.Errorf("Value ciao not present in the cold cache bitmap")
	}
	assertKeyInColdCache(t, c, "key1")

	// Put a line in the cache for key2, must be in the cold cache
	c.Put("key2", genBuffer("ciao2", 32))
	if c.cold.buckets[0].freecells.bit(62) != 1 {
		t.Errorf("Value ciao2 not present in the cold cache bitmap")
	}
	assertKeyInColdCache(t, c, "key2")

	// Put a line in the cache for key2, key2 must be moved to the hot cache
	// and the cold emptied for that key
	c.Put("key1", genBuffer("ciao", 32))
	if c.cold.buckets[0].freecells.bit(63) != 0 {
		t.Errorf("Value ciao still present in the cold cache bitmap")
	}
	assertKeyInHotCache(t, c, "key1")

	// Put a line that is bigger than the biggest cold cache bucket, so it has
	// to go in the hot cache.
	c.Put("key3", genBuffer("\x52\xfd\xfc\x07\x21\x82\x65\x4f\x16\x3f\x5f\x0f\x9a\x62\x1d\x72\x95\x66\xc7\x4d\x10\x03\x7c\x4d\x7b\xbb\x04\x07\xd1\xe2\xc6\x49\x81\x85\x5a\xd8\x68\x1d\x0d\x86\xd1\xe9\x1e\x00\x16\x79\x39\xcb\x66\x94\xd2\xc4\x22\xac\xd2\x08\xa0\x07\x29\x39\x48\x7f\x69\x99", 128))
	assertKeyInHotCache(t, c, "key3")

	c.Flush()

	if len(c.m.m) > 0 {
		t.Errorf("Didn't properly flush")
	}
	if !flushed {
		t.Errorf("got flushed = false, want true")
	}

	assertKeyNotInCache(t, c, "key1")
	assertKeyNotInCache(t, c, "key2")
	assertKeyNotInCache(t, c, "key3")
}

func TestBufferCache_PutAndFlush(t *testing.T) {
	flushed := false
	cfg := Config{
		MaxBufferLength:   128,
		MaxCapacity:       512,
		CellsPerBucket:    64,
		Buckets:           []int{64},
		OnFlush:           func([]byte) { flushed = true },
		EnableCompression: false,
	}

	c, _ := New(cfg)

	// Put a line in the cache for key1, must be in the cold cache
	c.Put("key1", []byte("ciao"))
	if c.cold.buckets[0].freecells.bit(63) != 1 {
		t.Errorf("Value ciao not present in the cold cache bitmap")
	}
	assertKeyInColdCache(t, c, "key1")

	// Put a line in the cache for key2, must be in the cold cache
	c.Put("key2", []byte("ciao2"))
	if c.cold.buckets[0].freecells.bit(62) != 1 {
		t.Errorf("Value ciao2 not present in the cold cache bitmap")
	}
	assertKeyInColdCache(t, c, "key2")

	// Put a line in the cache for key2, key2 must be moved to the hot cache
	// and the cold emptied for that key
	c.Put("key1", []byte("ciao"))
	if c.cold.buckets[0].freecells.bit(63) != 0 {
		t.Errorf("Value ciao still present in the cold cache bitmap")
	}
	assertKeyInHotCache(t, c, "key1")

	// Put a line that is bigger than the biggest cold cache bucket, so it has
	// to go in the hot cache.
	c.Put("key3", []byte("qwertyuiopqwertyuiopqwertyuiopqwertyuiopqwertyuiopqwertyuiopqwertyuiop"))
	assertKeyInHotCache(t, c, "key3")

	c.Flush()

	if len(c.m.m) > 0 {
		t.Errorf("Didn't properly flush")
	}
	if !flushed {
		t.Errorf("got flushed = false, want true")
	}

	assertKeyNotInCache(t, c, "key1")
	assertKeyNotInCache(t, c, "key2")
	assertKeyNotInCache(t, c, "key3")
}

func TestBufferCacheAutoFlushCold(t *testing.T) {
	// Fill the cold cache and check flush is called and that the locationMap
	// reflects the state of the cache.
	flushed := false
	cfg := Config{
		MaxBufferLength: 128,
		MaxCapacity:     512,
		CellsPerBucket:  64,
		Buckets:         []int{64},
		OnFlush:         func([]byte) { flushed = true },
	}

	c, _ := New(cfg)

	for i := 0; i < cfg.CellsPerBucket; i++ {
		c.Put(fmt.Sprintf("key%d", i), genBuffer("ciao", 32))
	}
	if flushed {
		t.Errorf("got flushed = true, want false")
	}
	c.Put("extrakey", genBuffer("ciao", 32))
	if !flushed {
		t.Errorf("got flushed = false, want true")
	}

	// check all keys are absent from the cache
	for i := 0; i < cfg.CellsPerBucket; i++ {
		key := fmt.Sprintf("key%d", i)
		assertKeyNotInCache(t, c, key)
	}

	assertKeyInColdCache(t, c, "extrakey")
}

func (c *BufferCache) cacheTotalize() int {
	coldCacheSize := 0
	for _, b := range c.cold.buckets {
		coldCacheSize += len(b.data)
	}
	return coldCacheSize + c.hot.cap
}

func TestBufferCache_size(t *testing.T) {
	cfg := Config{
		MaxBufferLength: 128,
		MaxCapacity:     10240,
		CellsPerBucket:  64,
		Buckets:         []int{64},
		OnFlush:         func([]byte) {},
	}
	c, _ := New(cfg)

	coldSize := cfg.CellsPerBucket * cfg.Buckets[0]

	if c.cacheTotalize() != coldSize {
		t.Errorf("got cacheTotalSize = %d, want %d", c.cacheTotalize(), coldSize)
	}

	// Adding to cold cache does not change the cache size
	c.Put("key1", genBuffer("ciao", 32))
	if c.cacheTotalize() != coldSize {
		t.Errorf("got cacheTotalSize=%v want %v", c.cacheTotalize(), coldSize)
	}

	c.Put("key2", genBuffer("ciao", 32))
	if c.cacheTotalize() != coldSize {
		t.Errorf("got cacheTotalSize=%v want %v", c.cacheTotalize(), coldSize)
	}

	// Adding a second buffer to the same key, however, does changes the size
	// since it gets appended to the previous buffer (they're both moved into the
	// hot cache).
	c.Put("key1", genBuffer("ciao2", 32))
	if c.cacheTotalize() < coldSize {
		t.Errorf("got cacheTotalSize=%v < %d, want >=", c.cacheTotalize(), coldSize)
	}
}

func TestBufferCacheMoveToHotCache(t *testing.T) {
	// Test the hash map is correctly updated when an adv is moved to hot cache
	cfg := Config{
		MaxBufferLength: 128,
		MaxCapacity:     10240,
		CellsPerBucket:  64,
		Buckets:         []int{64},
		OnFlush:         func([]byte) {},
	}
	c, _ := New(cfg)

	// Adding to cold cache must not change the size
	c.Put("key1", []byte("ciao"))
	if c.cacheTotalize() != 64*64 {
		t.Errorf("got cacheTotalSize=%v want %v", c.cacheTotalize(), 64*64)
	}

	assertKeyInColdCache(t, c, "key1")

	loc := c.m.m["key1"]
	c.moveToHotCache("key1", loc)
	assertKeyInHotCache(t, c, "key1")

	c.Flush()
	// After a flush, putting the key again should add it to the cold cache
	c.Put("key1", []byte("hello"))
	assertKeyInColdCache(t, c, "key1")
}

func TestBufferCacheFlushAutoHotOnMaxBufferLength(t *testing.T) {
	// Test the hash map is correctly updated when the hot cache is auto-flushed
	// after an insertion (because maxbuflen would have been reached)
	cfg := Config{
		MaxBufferLength: 127,
		MaxCapacity:     2560,
		CellsPerBucket:  64,
		Buckets:         []int{64},
		OnFlush:         func([]byte) {},
	}
	c, _ := New(cfg)

	// Can't stay in a 64b cell, goes directly in hot cache.
	c.Put("key1", incompressible[:100])
	assertKeyInHotCache(t, c, "key1")

	// This doesn't go in cold cache since key1 is already in the hot cache,
	// but this also can't immediately go in hot cache since the whole buffer
	// for that key would exceed maxBufferLength. So the hot[key1] entry is
	// flushed, and this goes into the cold cache.
	bbbb30 := incompressible[:30]
	c.Put("key1", bbbb30)
	assertKeyInColdCache(t, c, "key1")
	assertKeyValue(t, c, "key1", bbbb30)
	assertKeyNotInHotCache(t, c, "key1")
}

func TestBufferCacheFlushAutoHotOnMaxCapacity(t *testing.T) {
	// Test the hash map is correctly updated when the hot cache is auto-flushed
	// after an insertion (because maxcap would have been reached).

	t.Run("after flush, new entry goes in cold cache", func(t *testing.T) {
		cfg := Config{
			MaxBufferLength: 136,
			MaxCapacity:     137,
			CellsPerBucket:  64,
			Buckets:         []int{64},
			OnFlush:         func([]byte) {},
		}
		c, err := New(cfg)
		if err != nil {
			t.Fatal(err)
		}

		// Can't stay in a 64b cell, goes directly in hot cache.
		c.Put("key1", incompressible[:64])
		assertKeyInHotCache(t, c, "key1")

		// Can't stay in a 64b cell, goes directly in hot cache.
		c.Put("key2", incompressible[64:128])
		assertKeyInHotCache(t, c, "key2")

		// Already seen key1, so this should go in the hot cache, but adding this
		// would exceed the hot cache capacity, so the whole hot cache is flushed
		// and this will finally go into the cold cache (since it can fit there).
		cccc30 := incompressible[:30]
		c.Put("key1", cccc30)
		assertKeyInColdCache(t, c, "key1")
		assertKeyValue(t, c, "key1", cccc30)

		// Finally, check key2 is nowhere to be found
		assertKeyNotInCache(t, c, "key2")
	})

	t.Run("after flush, new entry goes in hot cache", func(t *testing.T) {
		// Test the hash map is correctly updated when the hot cache is auto-flushed
		// after an insertion (because maxcap would have been reached), but the new
		// entry is too big and goes directly in hot cache.
		cfg := Config{
			MaxBufferLength: 136,
			MaxCapacity:     137,
			CellsPerBucket:  64,
			Buckets:         []int{64},
			OnFlush:         func([]byte) {},
		}
		c, _ := New(cfg)

		// Can't stay in a 64b cell, goes directly in hot cache.
		c.Put("key1", incompressible[:64])
		assertKeyInHotCache(t, c, "key1")

		// Can't stay in a 64b cell, goes directly in hot cache.
		c.Put("key2", incompressible[64:128])
		assertKeyInHotCache(t, c, "key2")

		// Already seen key1, so this should go in the hot cache, but adding this
		// would exceed the hot cache capacity, so the whole hot cache is flushed.
		// Then the new entry goes directly in the hot cache because it doesn't
		// directly fit in the cold cache.
		cccc30 := incompressible[128:192]
		c.Put("key1", cccc30)
		assertKeyInHotCache(t, c, "key1")
		assertKeyValue(t, c, "key1", cccc30)

		// Finally, check key2 is nowhere to be found
		assertKeyNotInCache(t, c, "key2")
	})
}

func TestPutBufferBiggerThanBuckets(t *testing.T) {
	// Test that when putting buffers bigger than the biggest bucket they go in
	// the hot cache directly.

	cfg := Config{
		MaxBufferLength: 16,
		MaxCapacity:     256,
		CellsPerBucket:  64,
		Buckets:         []int{1},
		OnFlush:         func([]byte) {},
	}

	c, _ := New(cfg)
	for i := 0; i < 64; i++ {
		c.Put("key", []byte("buff"))
	}
}

func TestPutBufferBiggerThanMaxBufferLength(t *testing.T) {
	cache, _ := New(Config{
		MaxBufferLength: 16,
		MaxCapacity:     256,
		CellsPerBucket:  64,
		Buckets:         []int{1},
		OnFlush:         func([]byte) {},
	})
	assertPanics(t, func() {
		// Can't put buf nowhere
		buf := bytes.Repeat([]byte{'a'}, 17)
		cache.Put("key", buf)
	})
}

func TestInvalidConfig(t *testing.T) {
	t.Run("maxbuflen > maxcap", func(t *testing.T) {
		_, err := New(Config{
			MaxBufferLength: 2,
			MaxCapacity:     1,
		})
		assertErrInvalidConfig(t, err)
	})

	t.Run("maxbuflen < 0", func(t *testing.T) {
		_, err := New(Config{
			MaxBufferLength: -1,
			MaxCapacity:     0,
		})
		assertErrInvalidConfig(t, err)
	})

	t.Run("maxbuflen < 0", func(t *testing.T) {
		_, err := New(Config{
			MaxBufferLength: 0,
			MaxCapacity:     -1,
		})
		assertErrInvalidConfig(t, err)
	})

	t.Run("ncells = 0", func(t *testing.T) {
		_, err := New(Config{
			MaxBufferLength: 1,
			MaxCapacity:     2,
			CellsPerBucket:  0,
		})
		assertErrInvalidConfig(t, err)
	})

	t.Run("ncells not multiple of 64", func(t *testing.T) {
		_, err := New(Config{
			MaxBufferLength: 1,
			MaxCapacity:     2,
			CellsPerBucket:  63,
		})
		assertErrInvalidConfig(t, err)
	})
}

func TestHotCacheMetrics(t *testing.T) {
	buf := []byte("buff")

	cfg := Config{
		MaxBufferLength:   80,
		MaxCapacity:       256,
		CellsPerBucket:    64,
		Buckets:           []int{1},
		EnableCompression: false,
	}
	c, _ := New(cfg)

	// put same key multiple times but do not exceed any limit so no flush
	for i := 0; i < 10; i++ {
		c.Put("key", buf)
		m := c.Metrics()
		want := Metrics{
			Hot: hotCacheMetrics{
				TotalEntries: 1,
				TotalSize:    uint64((i + 1) * (4 + 4)),
			},
			Cold:         coldCacheMetrics{FillRatios: []float32{0}},
			TotalFlushes: 0,
		}
		if !reflect.DeepEqual(m, want) {
			t.Fatalf("after %dth put: got %+v, want %+v", i, m, want)
		}
	}

	// Trigger a flush by exceeding MaxBufferLength for "key"
	c.Put("key", buf)
	m := c.Metrics()
	want := Metrics{
		Hot: hotCacheMetrics{
			TotalEntries: 1,
			TotalSize:    8,
		},
		Cold:         coldCacheMetrics{FillRatios: []float32{0}},
		TotalFlushes: 1,
	}
	if !reflect.DeepEqual(m, want) {
		t.Fatalf("got %+v, want %+v", m, want)
	}

	// Put new keys but never exceed any limit
	c.Put("key1", make([]byte, 76))
	c.Put("key2", make([]byte, 76))
	c.Put("key3", make([]byte, 76))
	m = c.Metrics()
	want = Metrics{
		Hot: hotCacheMetrics{
			TotalEntries: 4,
			TotalSize:    248,
		},
		Cold:         coldCacheMetrics{FillRatios: []float32{0}},
		TotalFlushes: 1,
	}
	if !reflect.DeepEqual(m, want) {
		t.Fatalf("got %+v, want %+v", m, want)
	}

	// Trigger a flush by exceeding MaxCapacity
	for i := 0; i < 8; i++ {
		c.Put("key4", make([]byte, 20))
	}
	m = c.Metrics()
	want = Metrics{
		Hot: hotCacheMetrics{
			TotalEntries: 1,
			TotalSize:    48,
		},
		Cold:         coldCacheMetrics{FillRatios: []float32{0}},
		TotalFlushes: 7,
	}
	if !reflect.DeepEqual(m, want) {
		t.Fatalf("got %+v, want %+v", m, want)
	}
}

func TestFlushUncompressedBlocks(t *testing.T) {
	// We're testing we do not panic

	cfg := Config{
		MaxBufferLength:   32,
		MaxCapacity:       64,
		CellsPerBucket:    64,
		Buckets:           []int{1},
		EnableCompression: false,
	}
	c, _ := New(cfg)
	c.comp.buf = make([]byte, 16)
	c.decomp.buf = make([]byte, 16)

	c.Put("k1", []byte("12b buf   n1"))
	c.Put("k1", []byte("12b buf   n2"))
	c.Put("k1", []byte("12b buf   n3"))

	c.Put("k1", []byte("12b buf   n1"))
	c.Put("k1", []byte("12b buf   n2"))
	c.Put("k1", []byte("12b buf   n3"))
}
