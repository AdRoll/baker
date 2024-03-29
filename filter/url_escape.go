package filter

import (
	"fmt"
	"net/url"
	"sync/atomic"

	"github.com/AdRoll/baker"
	log "github.com/sirupsen/logrus"
)

var URLEscapeDesc = baker.FilterDesc{
	Name:   "URLEscape",
	New:    NewURLEscape,
	Config: &URLEscapeConfig{},
	Help:   "Escape/Unescape URL. Escaping always succeeds but unescaping may fail, in which case this filter clears the destination field.",
}

type URLEscapeConfig struct {
	SrcField string `help:"Name of the field with the URL to escape/unescape" required:"true"`
	DstField string `help:"Name of the field to write the escaped/unescaped URL to." required:"true"`
	Unescape bool   `help:"Unescape the field instead of escaping it." default:"false"`
}

type URLEscape struct {
	numFilteredLines int64
	src, dst         baker.FieldIndex
	process          func([]byte) []byte
}

func NewURLEscape(cfg baker.FilterParams) (baker.Filter, error) {
	if cfg.DecodedConfig == nil {
		log.Fatal("No configuration provided")
	}

	dcfg := cfg.DecodedConfig.(*URLEscapeConfig)

	src, ok := cfg.FieldByName(dcfg.SrcField)
	if !ok {
		return nil, fmt.Errorf("unknwon field, SrcField = %q", dcfg.SrcField)
	}

	dst, ok := cfg.FieldByName(dcfg.DstField)
	if !ok {
		return nil, fmt.Errorf("unknwon field, DstField = %q", dcfg.SrcField)
	}

	process := func(s []byte) []byte {
		return []byte(url.QueryEscape(string(s)))
	}
	if dcfg.Unescape {
		process = func(s []byte) []byte {
			u, err := url.QueryUnescape(string(s))
			if err != nil {
				return nil
			}

			return []byte(u)
		}
	}

	f := &URLEscape{
		src:     src,
		dst:     dst,
		process: process,
	}
	return f, nil
}

func (f *URLEscape) Stats() baker.FilterStats {
	return baker.FilterStats{
		NumFilteredLines: atomic.LoadInt64(&f.numFilteredLines),
	}
}

func (f *URLEscape) Process(l baker.Record, next func(baker.Record)) {
	l.Set(f.dst, f.process(l.Get(f.src)))
	next(l)
}
