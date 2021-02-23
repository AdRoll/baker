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
	Help: `Set the last modified timestamp of the underlaying data source of the log line
into a connfigurable Field.`,
}

type MetadataLastModifiedConfig struct {
	DstField string `help:"Name of the field into which write the timestamp" required:"true"`
}

type MetadataLastModified struct {
	numProcessedLines int64
	numFilteredLines  int64
	cfg               *MetadataLastModifiedConfig
	dst               baker.FieldIndex
}

func NewMetadataLastModified(cfg baker.FilterParams) (baker.Filter, error) {
	if cfg.DecodedConfig == nil {
		cfg.DecodedConfig = &MetadataLastModified{}
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
		NumFilteredLines:  atomic.LoadInt64(&f.numFilteredLines),
	}
}

func (f *MetadataLastModified) Process(l baker.Record, next func(baker.Record)) {
	v, ok := l.Meta(inpututils.MetadataLastModified)
	if ok {
		lastModified := v.(time.Time)
		if !lastModified.IsZero() {
			unixTime := lastModified.Unix()
			timestampStr := []byte(strconv.FormatInt(unixTime, 10))
			l.Set(f.dst, timestampStr)
			atomic.AddInt64(&f.numProcessedLines, 1)
		}
	}
	next(l)
}
