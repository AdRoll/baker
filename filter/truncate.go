package filter

import (
	"fmt"

	"github.com/AdRoll/baker"
)

var TruncateDesc = baker.FilterDesc{
	Name:   "Truncate",
	New:    NewTruncate,
	Config: &TruncateConfig{},
	Help:   "Truncates the source field value to the given bytes length and saves the value to the destination field",
}

type TruncateConfig struct {
	Src      string `help:"The source field to truncate" required:"true"`
	Dst      string `help:"The destination field to save the truncated value to" required:"true"`
	Length   int    `help:"The lenght of the truncation to apply to the source value" required:"true"`
	StartIdx int    `help:"The byte 0-based index to start truncating from" default:"0"`
}

// Truncate filter truncates the source field value to the given bytes length and saves the value to the destination field
type Truncate struct {
	src      baker.FieldIndex
	dst      baker.FieldIndex
	length   int
	startIdx int
}

// NewTruncate creates a new Truncate filter
func NewTruncate(cfg baker.FilterParams) (baker.Filter, error) {
	dcfg := cfg.DecodedConfig.(*TruncateConfig)

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

	f := &Truncate{
		src:      src,
		dst:      dst,
		length:   dcfg.Length,
		startIdx: dcfg.StartIdx,
	}

	return f, nil
}

// Process records, truncating src fields to the given length, saving the result to the dest field
func (f *Truncate) Process(r baker.Record, next func(baker.Record)) {
	src := r.Get(f.src)

	if f.startIdx >= len(src) {
		r.Set(f.dst, []byte(""))
		next(r)
		return
	}

	var end = f.startIdx + f.length
	if end > len(src) {
		end = len(src)
	}

	r.Set(f.dst, src[f.startIdx:end])
	next(r)
}

// Stats implements baker.Filter.
func (f *Truncate) Stats() baker.FilterStats {
	return baker.FilterStats{}
}
