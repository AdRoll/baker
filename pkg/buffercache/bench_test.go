package buffercache

import (
	"math/rand"
	"os"
	"strconv"
	"testing"
)

var keys [419]string

func TestMain(m *testing.M) {
	keybuf := make([]byte, 22)
	for i := range keys {
		rand.Read(keybuf)
		keys[i] = string(keybuf)
	}
	os.Exit(m.Run())
}

func BenchmarkHotPutNoFlush(b *testing.B) {
	b.Run("puts=1000", func(b *testing.B) { benchmarkHotPutNoFlush(b, 1000) })
	b.Run("puts=10000", func(b *testing.B) { benchmarkHotPutNoFlush(b, 10000) })
	b.Run("puts=100000", func(b *testing.B) { benchmarkHotPutNoFlush(b, 100000) })
	b.Run("puts=1000000", func(b *testing.B) { benchmarkHotPutNoFlush(b, 1000000) })
}

func benchmarkHotPutNoFlush(b *testing.B, nkeys int) {
	maxcap := 2 * 1024 * nkeys

	buf := make([]byte, 2*1024)
	cfg := Config{
		MaxBufferLength: maxcap,
		MaxCapacity:     maxcap,
		CellsPerBucket:  64,
		Buckets:         []int{1}, // so small that all goes in hot cache
	}

	cache, err := New(cfg)
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	b.SetBytes(int64(len(buf)))

	cap := 0
	for n := 0; n < b.N; n++ {
		cache.Put(keys[n%419], buf)
		cap += len(buf) + 1
		if cap+len(buf)+1 > maxcap {
			b.StopTimer()
			cache.Flush()
			cap = 0
			b.StartTimer()
		}
	}
}

func BenchmarkColdPutNoFlush(b *testing.B) {
	b.Run("buckets=1024", func(b *testing.B) { benchmarkColdPutNoFlush(b, 1024) })
	b.Run("buckets=2048", func(b *testing.B) { benchmarkColdPutNoFlush(b, 2048) })
	b.Run("buckets=4096", func(b *testing.B) { benchmarkColdPutNoFlush(b, 4096) })
	b.Run("buckets=8192", func(b *testing.B) { benchmarkColdPutNoFlush(b, 8192) })
}

func benchmarkColdPutNoFlush(b *testing.B, ncells int) {
	wantFlush := false
	flush := func([]byte) {
		if !wantFlush {
			b.Fatalf("got flushed")
		}
	}

	buf := make([]byte, 2*1024-4)
	cfg := Config{
		MaxBufferLength: len(buf),
		MaxCapacity:     len(buf),
		CellsPerBucket:  ncells,
		Buckets:         []int{256, 512, 1024, 2 * 1024}, // put some smaller buckets before
		OnFlush:         flush,
	}

	cache, err := New(cfg)
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	b.SetBytes(int64(len(buf)))

	for n := 0; n < b.N; n++ {
		cache.Put(strconv.Itoa(n), buf)
		if n%(ncells-1) == 0 {
			b.StopTimer()
			wantFlush = true
			cache.Flush()
			wantFlush = false
			b.StartTimer()
		}
	}
}
