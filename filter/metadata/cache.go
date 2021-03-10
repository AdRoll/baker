// Package metadata provides a metadata-specific (concurrent and size-limited) cache.
package metadata

import "sync"

const maxCacheEntries int64 = 100

// Cache is a concurrent size-limited cache. If the cache size reaches
// maxCacheEntries all existing entries are removed.
//
// maxCacheEntries should effectively be the number of workers in the
// topology, but rather than expose that information to filters we for
// now just use an estimated max of 100.
type Cache struct {
	cache sync.Map

	mx       sync.Mutex
	nentries int64
}

// Store caches key and value in the cache.
func (c *Cache) Store(key, value interface{}) {
	c.mx.Lock()

	if c.nentries == maxCacheEntries {
		c.nentries -= c.clear()
	}

	if _, loaded := c.cache.LoadOrStore(key, value); !loaded {
		c.nentries++
	}

	c.mx.Unlock()
}

// Load loads from the cache the value that is mapped to key, or bool if
// the cache doesn't contain key.
func (c *Cache) Load(key interface{}) (value interface{}, ok bool) {
	return c.cache.Load(key)
}

// clear must be called with lock already held!
func (c *Cache) clear() int64 {
	var i int64
	c.cache.Range(func(key, _ interface{}) bool {
		c.cache.Delete(key)
		i++
		return true
	})
	return i
}
