package filter

import (
	"fmt"
	"net/url"

	"github.com/AdRoll/baker"
)

const (
	urlParamHelp = `
This filter extracts a query parameter (Param) from a source field (SrcField)
containing a URL and saves it into a destination field (DstField).

Error handling:
- If "SrcField" does not contain a valid URL an empty string will be stored in
DstField.
- If the query param "Param" is not present in the URL an empty string will be
stored in DstField.
`
)

var URLParamDesc = baker.FilterDesc{
	Name:   "URLParam",
	New:    NewURLParam,
	Config: &URLParamConfig{},
	Help:   urlParamHelp,
}

type URLParamConfig struct {
	SrcField string `help:"field containing the url." required:"true"`
	DstField string `help:"field to save the extracted url param." required:"true"`
	Param    string `help:"name of the url parameter to extract." required:"true"`
}

type URLParam struct {
	srcField baker.FieldIndex
	dstField baker.FieldIndex
	param    string
}

func NewURLParam(cfg baker.FilterParams) (baker.Filter, error) {
	dcfg := cfg.DecodedConfig.(*URLParamConfig)

	srcFieldIdx, ok := cfg.FieldByName(dcfg.SrcField)
	if !ok {
		return nil, fmt.Errorf("URLParam: unknown field %s", dcfg.SrcField)
	}

	dstFieldIdx, ok := cfg.FieldByName(dcfg.DstField)
	if !ok {
		return nil, fmt.Errorf("URLParam: unknown field %s", dcfg.DstField)
	}

	return &URLParam{srcField: srcFieldIdx, dstField: dstFieldIdx, param: dcfg.Param}, nil
}

func (f *URLParam) Process(l baker.Record, next func(baker.Record)) {
	param := ""

	u, err := url.Parse(string(l.Get(f.srcField)))
	if err == nil {
		param = u.Query().Get(f.param)
	}

	l.Set(f.dstField, []byte(param))

	next(l)
}

func (c *URLParam) Stats() baker.FilterStats {
	return baker.FilterStats{}
}
