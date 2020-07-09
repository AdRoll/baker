package baker

// FieldIndex is the index uniquely representing of a field in a Record.
type FieldIndex int

// Record is the basic object being processed by baker components. types
// implementing Record hold the memory representation of a single record.
type Record interface {
	// Parse decodes a buffer representing a record in its data format into
	// the current record instance.
	//
	// The given Metadata will be attached to that record. Record
	// implementations should also accept a nil in case the record has no
	// Metadata attached.
	Parse([]byte, *Metadata) error

	// ToText returns the reconstructed data format of a record.
	//
	// In case a big enough buf is passed, it will be used to serialize the
	// record.
	ToText(buf []byte) []byte

	// Clear clears the record internal state, making it empty.
	Clear()

	// Get the value of a field.
	Get(FieldIndex) []byte

	// Set the value of a field.
	Set(FieldIndex, []byte)

	// Meta returns the value of the attached metadata for the given key, if any.
	//
	// Records implementers may implement that method by declaring:
	//  type MyRecord struct {
	// 	      meta baker.Metadata
	//  }
	//
	//  func (r *MyRecord) Meta(key string) (interface{}, bool) {
	//  	return l.meta.get(key)
	//  }
	Meta(key string) (v interface{}, ok bool)

	// Cache holds a cache which is local to the record. It may be used to
	// speed up parsing of specific fields by caching the result. When
	// accessing a field and parsing its value, we want to try caching as much
	// as possible the parsing we do, to avoid redoing it later when
	// the same record is processed by different code.
	// Since cached values are interfaces it's up to who fetches a value to
	// know the underlying type of the cached value and perform a type assertion.
	//
	//  var ll Record
	//  val, ok := ll.Cache.Get("mykey")
	//  if !ok {
	//  	// long computation/parsing...
	//  	val = "14/07/1789"
	//  	ll.Cache.Set("mykey", val)
	//  }
	//
	//  // do something with the result
	//  result := val.(string)
	Cache() *Cache
}

// Cache is a per-record cache.
type Cache map[string]interface{}

// Get fetches the value with the given key. If the key is not present
// in the cache Get returns (nil, false).
func (c *Cache) Get(key string) (val interface{}, ok bool) {
	if *c == nil {
		return nil, false
	}

	val, ok = (*c)[key]
	return
}

// Set assigns the given value to a specific key.
func (c *Cache) Set(key string, val interface{}) {
	if *c == nil {
		*c = Cache{}
	}

	(*c)[key] = val
}

// Del removes the given cache entry.
func (c *Cache) Del(key string) {
	if *c == nil {
		return
	}

	delete(*c, key)
}

// Clear clears all the entries in the cache.
func (c *Cache) Clear() {
	*c = nil
}
