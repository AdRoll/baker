package filter

import (
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/AdRoll/baker"
	"github.com/AdRoll/baker/input/inpututils"
)

var MetadataLastModifiedDesc = baker.FilterDesc{
	Name:   "MetadataLastModified",
	New:    NewMetadataLastModified,
	Config: &MetadataLastModifiedConfig{},
	Help:   `Extract the "last modified" timestamp from the record Metadata and write it to the selected field.`,
}

type MetadataLastModifiedConfig struct {
	DstField string `help:"Name of the field into which write the timestamp to" required:"true"`
}

type MetadataLastModified struct {
	cfg *MetadataLastModifiedConfig

	dst baker.FieldIndex

	// Shared state
	numProcessedLines int64
}

func NewMetadataLastModified(cfg baker.FilterParams) (baker.Filter, error) {
	if cfg.DecodedConfig == nil {
		cfg.DecodedConfig = &MetadataLastModifiedConfig{}
	}
	dcfg := cfg.DecodedConfig.(*MetadataLastModifiedConfig)

	f := &MetadataLastModified{cfg: dcfg}

	dst, ok := cfg.FieldByName(dcfg.DstField)
	if !ok {
		return nil, fmt.Errorf("can't find the DstField %q", dcfg.DstField)
	}
	f.dst = dst
	return f, nil
}

func (f *MetadataLastModified) Stats() baker.FilterStats {
	return baker.FilterStats{
		NumProcessedLines: atomic.LoadInt64(&f.numProcessedLines),
	}
}

func (f *MetadataLastModified) Process(l baker.Record, next func(baker.Record)) {
	atomic.AddInt64(&f.numProcessedLines, 1)

	v, ok := l.Meta(inpututils.MetadataLastModified)
	if ok {
		lastModified := v.(time.Time)
		if !lastModified.IsZero() {
			unixTime := lastModified.Unix()
			timestampStr := []byte(strconv.FormatInt(unixTime, 10))
			l.Set(f.dst, timestampStr)
		}
	}

	next(l)
}
