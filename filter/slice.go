package filter

import (
	"fmt"

	"github.com/AdRoll/baker"
)

var SliceDesc = baker.FilterDesc{
	Name:   "Slice",
	New:    NewSlice,
	Config: &SliceConfig{},
	Help:   "Slices the source field value to the given bytes length and start index and saves the value to the destination field. If the start index is greater than the field value lenght, set an empty string to destination",
}

type SliceConfig struct {
	Src      string `help:"The source field to slice" required:"true"`
	Dst      string `help:"The destination field to save the sliced value to" required:"true"`
	Length   int    `help:"The lenght of the truncation to apply to the source value" required:"true"`
	StartIdx int    `help:"The 0-based byte index to start slicing from" default:"0"`
}

// Slice filter slices the source field value to the given bytes length and saves the value to the destination field
type Slice struct {
	src      baker.FieldIndex
	dst      baker.FieldIndex
	length   int
	startIdx int
}

// NewSlice creates a new Slice filter
func NewSlice(cfg baker.FilterParams) (baker.Filter, error) {
	dcfg := cfg.DecodedConfig.(*SliceConfig)

	src, ok := cfg.FieldByName(dcfg.Src)
	if !ok {
		return nil, fmt.Errorf("cannot find source field \"%s\"", dcfg.Src)
	}

	dst, ok := cfg.FieldByName(dcfg.Dst)
	if !ok {
		return nil, fmt.Errorf("cannot find destination field \"%s\"", dcfg.Dst)
	}

	if dcfg.Length < 1 {
		return nil, fmt.Errorf("invalid length %d", dcfg.Length)
	}

	f := &Slice{
		src:      src,
		dst:      dst,
		length:   dcfg.Length,
		startIdx: dcfg.StartIdx,
	}

	return f, nil
}

// Process records, slicing src fields to the given length, saving the result to the dest field
func (f *Slice) Process(r baker.Record, next func(baker.Record)) {
	v := r.Get(f.src)
	l := len(v)

	if f.startIdx >= l {
		r.Set(f.dst, []byte(""))
		next(r)
		return
	}

	end := f.startIdx + f.length
	if end > l {
		end = l
	}

	r.Set(f.dst, v[f.startIdx:end])
	next(r)
}

// Stats implements baker.Filter.
func (f *Slice) Stats() baker.FilterStats {
	return baker.FilterStats{}
}
