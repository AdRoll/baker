package baker

import "errors"

const (
	// LogLineNumFields is the maximum number of standard fields in a log line.
	// This is also the maximum number of field separators, a valid log line can
	// have also a tralling separator.
	LogLineNumFields FieldIndex = 3000
	// NumFieldsBaker is an additional list of custom fields, not present
	// in the input logline nor in the output, that can be set during processing.
	// Its main purpose it to fastly exchange values between filters (and possibly
	// outputs) on a per-record basis.
	NumFieldsBaker FieldIndex = 100

	// DefaultLogLineFieldSeparator defines the default field separator, which is the comma.
	DefaultLogLineFieldSeparator byte = 44
)

// LogLine represents a CSV text line using ASCII 30 as field separator. It
// implement Record..
//
// In memory, it is kept in a format optimized for very fast parsing and low
// memory-consumption. The vast majority of fields are never accessed during the
// lifetime of an object, as a filter usually reads or writes just a handful of
// fields; it thus makes sense to do the quickest possible initial parsing,
// deferring as much as possible to when a field is actually accessed.
//
// It is also possible to modify a LogLine in memory, as it gets processed.
// Modifications can be done through the Set() method, and can be done to any
// field, both those that had a parsed value, and those that were empty.
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

	// meta values can be filled in by the input to add informations on the
	// datasource of the Logline, like timestamps, originating S3 file,
	// debugging info or other metadata.  Values can be accessed by filters or
	// output to perform checks, transformations, etc.
	meta Metadata

	// This triplet handles in-memory modifications to LogLines (through
	// LogLine.Set()).
	// wcnt is the 1-based counter of how many fields were modified;
	// wdata is the dense storage for those modifications (so we allow for a
	// total of 254 different fields being written to).
	// wmask is a table indexed by each possible field index, that contains:
	//   * 0 if the field was not modified (so the current value can be fetched
	//     by idx/data)
	//   * the index into wdata were the new value for the field is stored (if
	//     the field was modified)
	//
	// NOTE: wdata[0] is never written to, because the index "0" in wmask is the
	// special value to signal "no modifications". We keep it like this because
	// we like that the zero-initialization of wmask does the right thing
	// (i.e. indicates that no fields have been written to).
	wmask [LogLineNumFields + NumFieldsBaker]uint8
	wdata [256][]byte
	wcnt  uint8

	cache Cache

	// FieldSeparator is the byte used to separate fields value.
	FieldSeparator byte
}

// Get the value of a field (either standard or custom).
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

// Set changes the value of a field (either standard or custom) to a new value.
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

var errLogLineTooManyFields = errors.New("LogLine has too many fields")

// Parse finds the next newline in data and parse log line fields from it into
// the current LogLine.
//
// This is the moral equivalent of bytes.Split(), but without memory allocations.
//
// NOTE: this function is meant to be called onto a just-constructed LogLine
// instance. For performance reasons, it doesn't reset all the writable fields
// of the line. If you want to use Parse over an already parsed LogLine, use
// Clear before.
func (l *LogLine) Parse(text []byte, meta Metadata) error {
	l.idx[0] = -1
	fc := FieldIndex(1)
	for i, ch := range text {
		if ch == l.FieldSeparator {
			// We return an error if we reach the last 'idx' array position.
			// Log lines with trailing separator are consider valid.
			if fc > LogLineNumFields {
				return errLogLineTooManyFields
			}
			l.idx[fc] = int32(i)
			fc++
		}
	}

	// Truncate the buffer after the last valid field, if we are parsing a log line
	// with a trailing separator. In the other case set the length of the buffer
	// as the last value and leave the rest of the array zeroed.
	if fc > LogLineNumFields {
		text = text[:l.idx[fc-1]]
	} else {
		l.idx[fc] = int32(len(text))
	}

	l.data = text
	if meta != nil {
		l.meta = meta
	}

	return nil
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
			// Not enough capacity: create a new buffer big enough to hold the
			// previous buffer data, which we copy into, and the log line data.
			newbuf := make([]byte, blen+dlen)
			copy(newbuf, buf)
			buf = newbuf
		} else {
			// We have the capacity, just reslice to the desired length.
			buf = buf[:blen+dlen]
		}
		copy(buf[blen:], l.data)
		return buf
	}

	// Get the last setted index in the write array.
	var last int
	for i := int(LogLineNumFields) - 1; i > 0; i-- {
		if l.wmask[i] != 0 {
			last = i
			break
		}
	}

	// Get the last index in the data buffer.
	if l.data != nil {
		var lastr int
		for i := len(l.idx) - 1; i > 0; i-- {
			if l.idx[i] != 0 {
				lastr = i - 1
				break
			}
		}
		// Update last value.
		if last < lastr {
			last = lastr
		}
	}

	// Compute an estimate of the max capacity required, so only one
	// allocation will ever be performed.
	var wlen int
	for i := uint8(1); i <= l.wcnt; i++ {
		wlen += len(l.wdata[i])
	}
	wlen += int(l.wcnt) - 1 // Add 1 additional byte per separator.

	blen, bcap, dlen := len(buf), cap(buf), len(l.data)
	avail := bcap - blen
	if avail < wlen+dlen {
		newbuf := make([]byte, blen, blen+dlen+wlen)
		copy(newbuf, buf)
		buf = newbuf
	}

	for fc := FieldIndex(0); fc < LogLineNumFields; fc++ {
		buf = append(buf, l.Get(fc)...)
		if fc >= FieldIndex(last) {
			break
		}
		buf = append(buf, l.FieldSeparator)
	}
	return buf
}

// Clear clears the logline.
func (l *LogLine) Clear() {
	*l = LogLine{FieldSeparator: l.FieldSeparator}
}

// Meta returns the metadata having the given specific key, if any.
func (l *LogLine) Meta(key string) (interface{}, bool) {
	return l.meta.get(key)
}

// Cache returns the cache that is local to the current log line.
func (l *LogLine) Cache() *Cache {
	return &l.cache
}

// Copy creates and returns a copy of the current log line.
func (l *LogLine) Copy() Record {
	// Copy metadata.
	md := make(Metadata)
	for k, v := range l.meta {
		md[k] = v
	}

	cpy := &LogLine{
		cache:          l.cache,
		meta:           md,
		FieldSeparator: l.FieldSeparator,
	}

	if l.wcnt != 0 {
		// If the log line has been modified, benchmarks have proven that it's
		// more efficient to serialize and reparse to perform a copy (both in
		// terms of time and allocation). Also, different benchmarks have shown
		// that pre-allocating 120% of the original log line length in order to
		// account for the potentially added fields is reasonable.
		cpylen := len(l.data) + len(l.data)/5
		text := l.ToText(make([]byte, 0, cpylen))
		cpy.Parse(text, md)
		return cpy
	}

	// If the log line hasn't been modified it's more efficient to recreate it
	// from scratch and copying data (log line internal buffer).
	if l.data != nil {
		cpy.data = make([]byte, len(l.data))
		copy(cpy.data, l.data)
		cpy.idx = l.idx
	}
	return cpy
}
