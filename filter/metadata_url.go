package filter

import (
	"fmt"
	"net/url"
	"sync/atomic"

	"github.com/AdRoll/baker"
	"github.com/AdRoll/baker/filter/metadata"
	"github.com/AdRoll/baker/input/inpututils"
)

var MetadataUrlDesc = baker.FilterDesc{
	Name:   "MetadataUrl",
	New:    NewMetadataUrl,
	Config: &MetadataUrlConfig{},
	Help:   `Extract the Metadata URL from the record Metadata and write it to the selected field`,
}

type MetadataUrlConfig struct {
	DstField string `help:"Name of the field into which write the timestamp to" required:"true"`
}

type MetadataUrl struct {
	cfg *MetadataUrlConfig

	dst baker.FieldIndex

	// Shared state
	numProcessedLines int64
	numFilteredLines  int64
	metadata.Cache    // type: map[*url.URL]string
}

func NewMetadataUrl(cfg baker.FilterParams) (baker.Filter, error) {
	if cfg.DecodedConfig == nil {
		cfg.DecodedConfig = &MetadataUrlConfig{}
	}
	dcfg := cfg.DecodedConfig.(*MetadataUrlConfig)

	f := &MetadataUrl{cfg: dcfg}

	dst, ok := cfg.FieldByName(dcfg.DstField)
	if !ok {
		return nil, fmt.Errorf("unknown field %q", dcfg.DstField)

	}
	f.dst = dst
	return f, nil
}

func (f *MetadataUrl) Stats() baker.FilterStats {
	return baker.FilterStats{
		NumProcessedLines: atomic.LoadInt64(&f.numProcessedLines),
		NumFilteredLines:  atomic.LoadInt64(&f.numFilteredLines),
	}
}

func (f *MetadataUrl) Process(l baker.Record, next func(baker.Record)) {
	v, ok := l.Meta(inpututils.MetadataURL)
	if ok {
		url := v.(*url.URL)
		if url != nil {
			urlStr, ok := f.Load(url)
			if !ok {
				urlStr = url.String()
				f.Store(url, urlStr)
			}
			l.Set(f.dst, []byte(urlStr.(string)))
			atomic.AddInt64(&f.numProcessedLines, 1)
		}
	}
	next(l)
}
