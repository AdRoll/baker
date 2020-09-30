package buffercache

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func assertErrInvalidConfig(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatalf("go err = nil, want errInvalidConfig")
	}
	if _, ok := err.(errInvalidConfig); !ok {
		t.Fatalf("go err = %v, want errInvalidConfig", err)
	}
}

func assertPanics(t *testing.T, f func()) {
	t.Helper()

	defer func() {
		if r := recover(); r != nil {
			return
		}
		t.Fatalf("didn't panic")
	}()
	f()
}

func assertKeyValue(t *testing.T, cache *BufferCache, key string, want []byte) {
	t.Helper()

	loc, ok := cache.m.m[key]
	if !ok {
		t.Fatalf(`key "%s" not found in location map`, key)
	}

	var (
		buf        []byte
		compressed bool
	)
	switch {
	case loc.isCold():
		bidx, cidx := loc.coldBucketIdx(), loc.coldCellIdx()
		buf, compressed = cache.cold.buckets[bidx].get(cidx)
	case loc.isHot():
		buf = cache.hot.m[key]

		// Ensure there's only one buffer
		prefix := binary.LittleEndian.Uint32(buf[:4])
		buflen := prefix & 0x7fffffff
		compressed = (prefix & 0x80000000) != 0
		buf = buf[4:]
		if int(buflen) != len(buf) {
			t.Fatalf("assertKeyValue only compares single buffer: got buflen != len(buf[4:])")
		}
	}

	if compressed {
		buf = cache.decomp.uncompressSimple(buf)
	}
	if !bytes.Equal(buf, want) {
		t.Errorf(`buffers don't match for key "%s", got = %s, want %s`, key, buf, want)
	}
}

func assertKeyNotInCache(t *testing.T, cache *BufferCache, key string) {
	t.Helper()

	loc, ok := cache.m.m[key]
	if ok {
		t.Errorf(`key "%s" found in cache at location=%v, should not be in cache`, key, loc)
	}
}

func assertKeyInHotCache(t *testing.T, cache *BufferCache, key string) {
	t.Helper()

	loc, ok := cache.m.m[key]
	if !ok {
		t.Fatalf(`key "%s" not found in location map`, key)
	}
	if !loc.isHot() {
		t.Errorf("not in hot cache: got loc=%v", loc)
	}
	if _, ok := cache.hot.m[key]; !ok {
		t.Errorf(`key "%s" not found in hot cache`, key)
	}
}

func assertKeyNotInHotCache(t *testing.T, cache *BufferCache, key string) {
	t.Helper()

	loc, ok := cache.m.m[key]
	if ok && loc.isHot() {
		t.Errorf(`got key "%s" in hot cache, got loc=%s`, key, loc)
	}
	if _, ok := cache.hot.m[key]; ok {
		t.Errorf(`key "%s" found in hot cache`, key)
	}
}

func assertKeyInColdCache(t *testing.T, cache *BufferCache, key string) {
	t.Helper()

	loc, ok := cache.m.m[key]
	if !ok {
		t.Fatalf(`key "%s" not found in location map`, key)
	}
	if !loc.isCold() {
		t.Errorf("not in cold cache: got loc=%v", loc)
	}
}

func assertColdLocation(t *testing.T, loc location, bidx, cidx int) {
	t.Helper()

	if !loc.isCold() {
		t.Errorf("not in cold cache: got loc=%v", loc)
	}
	if bidx != loc.coldBucketIdx() {
		t.Errorf("wrong bucket: got bidx=%d, want %d", loc.coldBucketIdx(), bidx)
	}
	if cidx != loc.coldCellIdx() {
		t.Errorf("wrong cell: got cidx=%d, want %d", loc.coldCellIdx(), cidx)
	}
}
