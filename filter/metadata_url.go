package filter

import (
	"fmt"
	"net/url"

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
	metadata.Cache // type: map[*url.URL]string
}

func NewMetadataUrl(cfg baker.FilterParams) (baker.Filter, error) {
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
	return baker.FilterStats{}
}

func (f *MetadataUrl) Process(l baker.Record, next func(baker.Record)) {
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
