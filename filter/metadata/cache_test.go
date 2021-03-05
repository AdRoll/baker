package metadata

import (
	"sync"
	"testing"
)

func TestCache(t *testing.T) {
	t.Parallel()

	c := Cache{}
	wg := sync.WaitGroup{}

	for i := 0; i < 100; i++ {
		wg.Add(1)
		j := i
		go func() {
			c.Store(j, j)
			wg.Done()
		}()
	}
	wg.Wait()

	if c.nentries != 100 {
		t.Errorf("unexpected cache size; got %d, want 100", c.nentries)
	}

	c.Store(100, 100)
	if c.nentries != 1 {
		t.Error("cache should have been cleared before adding next item")
	}
	_, ok := c.Load(100)
	if !ok {
		t.Error("item 100 should have been found in cache")
	}

	c.Store(100, 101)
	v, ok := c.Load(100)
	if !ok || c.nentries != 1 {
		t.Error("storing duplicate item should not have cleared cache or changed count")
	}
	if v != 100 {
		t.Errorf("got c.Load(100) = %v, want 100", v)
	}
}
