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
- If "SrcField" does not contain a valid URL, an ErrURLParamInvalidURL error is triggered.
- If the query param "Param" is not found in the URL, ErrURLParamNotFound is triggered.
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

func (f *URLParam) Process(l baker.Record) error {
	ustr := string(l.Get(f.srcField))
	u, err := url.Parse(ustr)
	if err != nil {
		return ErrURLParamInvalidURL{url: ustr, err: err}
	}

	if !u.Query().Has(f.param) {
		return ErrURLParamNotFound(f.param)
	}

	l.Set(f.dstField, []byte(u.Query().Get(f.param)))
	return nil
}

func (c *URLParam) Stats() baker.FilterStats {
	return baker.FilterStats{}
}

// wraps the error returned by url.Parse
type ErrURLParamInvalidURL struct {
	url string
	err error
}

func (e ErrURLParamInvalidURL) Error() string {
	return fmt.Sprintf("invalid url %v: %v", e.url, e.err)
}

type ErrURLParamNotFound string

func (e ErrURLParamNotFound) Error() string {
	return fmt.Sprintf("url param %v not found", string(e))
}
