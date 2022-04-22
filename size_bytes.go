package baker

import (
	"fmt"
	"math"

	"github.com/dustin/go-humanize"
)

type SizeBytes uint64

func (b *SizeBytes) UnmarshalTOML(p interface{}) error {
	var actual uint64

	switch v := p.(type) {
	case float64:
		if v < 0 {
			return fmt.Errorf("invalid size in bytes(%v): value must >= 0", v)
		}
		if v >= math.MaxUint64 {
			return fmt.Errorf("invalid size in bytes (%v): value must be smaller than %v", v, uint64(math.MaxUint64))
		}
		actual = uint64(v)
	case int64:
		if v < 0 {
			return fmt.Errorf("invalid size in bytes (%v): value must be >= 0", v)
		}
		actual = uint64(v)
	case string:
		if v != "" {
			var err error
			actual, err = humanize.ParseBytes(v)
			if err != nil {
				return fmt.Errorf("invalid size in bytes (%v): %v", v, err)
			}
		}
	default:
		return fmt.Errorf("unexpected type (%T): unexpected value type", v)
	}
	*b = SizeBytes(actual)
	return nil
}
