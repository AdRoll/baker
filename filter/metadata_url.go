package filter

import (
	"fmt"
	"net/url"
	"sync/atomic"

	"github.com/AdRoll/baker"
	"github.com/AdRoll/baker/filter/metadata"
	"github.com/AdRoll/baker/input/inpututils"
)

const metadataUrlHelp = `
This filter looks for 'url' in records metadata and copies it into a field of your choice, see DstField.
If it doesn't find the 'url' in the metadata, this filter clear DstField.

If you wish to discard records without the 'url' metadata, you can add the NotNull filter after this one in your topology.
`

var MetadataUrlDesc = baker.FilterDesc{
	Name:   "MetadataUrl",
	New:    NewMetadataUrl,
	Config: &MetadataUrlConfig{},
	Help:   metadataUrlHelp,
}

type MetadataUrlConfig struct {
	DstField string `help:"Name of the field into to write the url to (or to clear if there's no url)" required:"true"`
}

type MetadataUrl struct {
	cfg *MetadataUrlConfig

	dst baker.FieldIndex

	// Shared state
	numProcessedLines int64
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
	}
}

func (f *MetadataUrl) Process(l baker.Record, next func(baker.Record)) {
	atomic.AddInt64(&f.numProcessedLines, 1)

	v, ok := l.Meta(inpututils.MetadataURL)
	if !ok {
		l.Set(f.dst, nil)
		next(l)
		return
	}

	url := v.(*url.URL)
	if url == nil {
		l.Set(f.dst, nil)
		next(l)
		return
	}

	urlStr, ok := f.Load(url)
	if !ok {
		urlStr = url.String()
		f.Store(url, urlStr)
	}
	l.Set(f.dst, []byte(urlStr.(string)))
	next(l)
}
