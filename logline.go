package baker

type FieldIndex int

const (
	// LogLineNumFields is the maximum number of fields in a log line.
	// TODO[opensource]: solve this before releasing baker-core as open-source?
	// how? pass it from outside? allocate idx and wmask dynamically?
	LogLineNumFields FieldIndex = 3000
	// original forklift.NumFieldsBaker depends on custom_params.json and it is 82 at the moment.
	// TODO[opensource]: evaluate impact on wmask with a custom number
	// we probably want to keep this for adroll-baker but not in the core.
	// how one uses the field indices is up to them.
	NumFieldsBaker FieldIndex = 100
)

// LogLine is the basic object being processed by a filter. Natively,
// it is a CSV text line using ASCII 30 as separator of the fields.
//
// In memory, it is kept in a format optimized for very fast parsing
// and low memory-consumption. The vast majority of fields are never
// accessed during the lifetime of an object, as a filter usually reads
// or writes just a handful of fields; it thus makes sense to do the
// quickest possible initial parsing, deferring as much as possible to
// when a field is actually accessed.
//
// It is also possible to modify a LogLine in memory, as it gets
// processed. Modifications can be done through the Set() method,
// and can be done to any field, both those that had a parsed value,
// and those that were empty.
//
// Normally, LogLine will be constructed through NewLogLineFromText(),
// that parses a CSV line (with separator 0x1E). But notice that
// a zero-init LogLine is a perfectly valid empty object, and can be
// used as such to contruct loglines starting from empty.
type LogLine struct {
	// These next few fields handle the read-only fields that were parsed from a
	// text logline. data is the original line in memory, while idx is the index
	// into the original line to the separator that lies before the beginning of
	// each field (idx[0] is always -1).  meta is the metadata associated with
	// the original input.
	// Note that data is never modified, because it would be very slow to do it
	// in-place, enlarging / shrinking fields as necessary; if the user code
	// wants to modify a field through Set(), it is stored in a parallel
	// data-structure (see wmask/wdata/wcnt below).
	idx  [LogLineNumFields + 1]int32
	data []byte

	// meta values can be filled in by the input to add informations on the datasource
	// of the Logline, like timestamps, originating S3 file, debugging info or other metadata.
	// Values can be accessed by filters or output to perform checks, transformations, etc.
	meta Metadata

	// This triplet handles in-memory modifications to LogLines (through LogLine.Set()).
	// wcnt is the 1-based counter of how many fields were modified; wdata is the dense
	// storage for those modifications (so we allow for a total of 256 different fields
	// being written to). wmask is a table indexed by each possible field index, that contains:
	//   * 0 if the field was not modified (so the current value can be fetched by idx/data)
	//   * the index into wdata were the new value for the field is stored (if the field was modified)
	//
	// NOTE: wdata[0] is never written to, because the index "0" in wmask is the special
	// value to signal "no modifications". We keep it like this because we like that the
	// zero-initialization of wmask does the right thing (= indicates that no fields have
	// been written to).
	wmask [LogLineNumFields + NumFieldsBaker]uint8
	wdata [256][]byte
	wcnt  uint8

	// Cache holds a cache which is local to the LogLine. It may be used to
	// speed up parsing of specific fields by caching the result. When
	// accessing a field and parsing its value, we want to try caching as much
	// as possibly the parsing we do, to avoid redoing it later on when
	// the same logline is processed by different code.
	// Since cached values are interfaces it's up to the who fetches a value to
	// know the underlying type of the cached value and performs a cast.
	//
	//  var ll LogLine
	//  val, ok := ll.Cache.Get("mykey")
	//  if !ok {
	//  	// long computation/parsing...
	//  	val = "14/07/1789"
	//  	ll.Cache.Set("mykey", val)
	//  }
	//
	//  // do something with the result
	//  result := val.(string)
	Cache cache
}

// Get the value of a field (either standard or custom)
func (l *LogLine) Get(f FieldIndex) []byte {
	if idx := l.wmask[f]; idx != 0 {
		return l.wdata[idx]
	}
	if f >= LogLineNumFields {
		return nil
	}

	s := l.idx[f] + 1
	e := l.idx[f+1]
	if e-s < 1 {
		return nil
	}
	return l.data[s:e]
}

// Set changes the value of a field (either standard or custom) to a new value
func (l *LogLine) Set(f FieldIndex, data []byte) {
	if l.wmask[f] != 0 {
		l.wdata[l.wmask[f]] = data
		return
	}
	l.wcnt++
	if l.wcnt == 0 {
		panic("too many fields changed")
	}
	l.wmask[f] = l.wcnt
	l.wdata[l.wcnt] = data
}

// NewLogLineFromText creates a LogLine from a line of text (parsing the TSV)
// This is the moral equivalent of bytes.Split(), but without memory allocations
func NewLogLineFromText(text []byte) (l LogLine) {
	l.Parse(text, nil)
	return
}

// Parse finds the next newline in data and parse log line fields from it
// into the current LogLine.
//
// This is the moral equivalent of bytes.Split(), but without memory allocations
// NOTE: this function is meant to be called onto a just-constructed
// LogLine instance. For performance reasons, it doesn't reset all
// the writable fields of the line.
func (l *LogLine) Parse(text []byte, meta *Metadata) {
	l.idx[0] = -1
	fc := FieldIndex(1)
	for i, ch := range text {
		if ch == 30 {
			if fc > LogLineNumFields {
				// This log line has more fields than expected, zero it out
				// to ensure it wouldn't pass validation.
				*l = LogLine{}
				return
			}
			l.idx[fc] = int32(i)
			fc++
		}
	}
	for ; fc <= LogLineNumFields; fc++ {
		l.idx[fc] = int32(len(text))
	}
	l.data = text
	if meta != nil {
		l.meta = *meta
	}
}

// ToText converts back the LogLine to textual format and appends it to
// the specified buffer.
// If called on a default constructed LogLine (zero-value), ToText
// returns nil, which is an useless but syntactically valid buffer.
func (l *LogLine) ToText(buf []byte) []byte {
	// Fast path: if no fields have been written, we can just copy the
	// content of the original buffer and return it.
	if l.wcnt == 0 {
		blen, bcap, dlen := len(buf), cap(buf), len(l.data)
		avail := bcap - blen
		if avail < dlen {
			// not enough capacity: create a new buffer big enough to hold the
			// previous buffer data, which we copy into, and the log line data.
			newbuf := make([]byte, blen+dlen)
			copy(newbuf, buf)
			buf = newbuf
		} else {
			// we have the capacity, just reslice to the desired length.
			buf = buf[:blen+dlen]
		}
		copy(buf[blen:], l.data)
		return buf
	}

	var lastw int
	for i := len(l.wmask) - 1; i > 0; i-- {
		if l.wmask[i] != 0 {
			lastw = i
			break
		}
	}

	// Compute an estimate of the max capacity required, so only one
	// allocation will ever be performed.
	var wlen int
	for i := uint8(1); i <= l.wcnt; i++ {
		wlen += len(l.wdata[i])
	}

	blen, bcap, dlen := len(buf), cap(buf), len(l.data)
	avail := bcap - blen
	if avail < wlen+dlen {
		newbuf := make([]byte, blen, blen+dlen+wlen)
		copy(newbuf, buf)
		buf = newbuf
	}

	done := false
	for fc := FieldIndex(0); fc < LogLineNumFields && !done; fc++ {
		buf = append(buf, l.Get(fc)...)
		buf = append(buf, 30)
		done = fc > FieldIndex(lastw) && (l.data == nil || l.idx[fc] == -1)
	}
	return buf
}

// Meta returns the metadata having the given specific key, if any.
func (l *LogLine) Meta(key string) (interface{}, bool) {
	return l.meta.get(key)
}

type cache map[string]interface{}

// Get fetches the value with the given key. If the key is not present
// in the cache Get returns nil, false.
func (c *cache) Get(key string) (val interface{}, ok bool) {
	if *c == nil {
		return nil, false
	}

	val, ok = (*c)[key]
	return
}

// Set assigns the given value to a specific key.
func (c *cache) Set(key string, val interface{}) {
	if *c == nil {
		*c = cache{}
	}

	(*c)[key] = val
}

// Del removes the given cache entry.
func (c *cache) Del(key string) {
	if *c == nil {
		return
	}

	delete(*c, key)
}

// Clear clears all the entries in the cache.
func (c *cache) Clear() {
	*c = nil
}

// NewLogLineFromMap populates an LogLine with the fields in m. Useful for testing purposes
func NewLogLineFromMap(m map[FieldIndex]string) LogLine {
	ll := LogLine{}
	for fidx, v := range m {
		if v != "" {
			ll.Set(fidx, []byte(v))
		}
	}
	return ll
}
