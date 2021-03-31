package filter

import (
	"fmt"

	"github.com/AdRoll/baker"
)

var SliceDesc = baker.FilterDesc{
	Name:   "Slice",
	New:    NewSlice,
	Config: &SliceConfig{},
	Help:   "Slices the source field value using start/end indexes and copies the value to the destination field. If the start index is greater than the field value length, set an empty string to destination",
}

type SliceConfig struct {
	Src      string `help:"The source field to slice" required:"true"`
	Dst      string `help:"The destination field to save the sliced value to" required:"true"`
	StartIdx int    `help:"The index representing the start of the slicing" default:"0"`
	EndIdx   int    `help:"The index representing the end of the slicing. Defaults to the end of the field"`
}

// Slice filter slices the source field value to the given bytes length and saves the value to the destination field
type Slice struct {
	src      baker.FieldIndex
	dst      baker.FieldIndex
	startIdx int
	endIdx   int
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

	if dcfg.EndIdx <= dcfg.StartIdx && dcfg.EndIdx > 0 {
		return nil, fmt.Errorf("end index must be greater than start index %d - %d", dcfg.StartIdx, dcfg.EndIdx)
	}

	f := &Slice{
		src:      src,
		dst:      dst,
		startIdx: dcfg.StartIdx,
		endIdx:   dcfg.EndIdx,
	}

	return f, nil
}

// Process records, slicing src field and saving the result to the dest field
func (f *Slice) Process(r baker.Record, next func(baker.Record)) {
	src := r.Get(f.src)

	end := f.endIdx
	if end > len(src) || end == 0 {
		end = len(src)
	}

	sz := end - f.startIdx
	if f.startIdx >= len(src) {
		sz = 0
	}

	sl := make([]byte, sz)

	if f.startIdx < len(src) {
		copy(sl, src[f.startIdx:end])
	}

	r.Set(f.dst, sl)
	next(r)
}

// Stats implements baker.Filter.
func (f *Slice) Stats() baker.FilterStats {
	return baker.FilterStats{}
}
