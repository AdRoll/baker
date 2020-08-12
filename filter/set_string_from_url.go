package filter

import (
	"bytes"
	"fmt"
	"net/url"
	"sync/atomic"

	"github.com/AdRoll/baker"
	"github.com/AdRoll/baker/input/inpututils"
	log "github.com/sirupsen/logrus"
)

// SetStringFromURLDesc describes the SetStringFromURL filter.
var SetStringFromURLDesc = baker.FilterDesc{
	Name:   "SetStringFromURL",
	New:    newSetStringFromURL,
	Config: &setStringFromURLConfig{},
	Help:   `Extract some strings from metadata url and sets a field with it.`,
}

type setStringFromURLConfig struct {
	Field   string   `help:"Name of the field to set to" required:"true"`
	Strings []string `help:"Strings to look for in the URL. Discard records not containing any of them." required:"true"`
}

func (cfg *setStringFromURLConfig) fillDefaults() error {
	if cfg.Field == "" {
		return fmt.Errorf("Field is a required parameter")
	}

	if len(cfg.Strings) == 0 {
		return fmt.Errorf("Strings is a required parameter")
	}

	return nil
}

type setStringFromURL struct {
	numProcessedLines int64
	numFilteredLines  int64

	field   baker.FieldIndex
	strings [][]byte
}

func newSetStringFromURL(cfg baker.FilterParams) (baker.Filter, error) {
	if cfg.DecodedConfig == nil {
		cfg.DecodedConfig = &setStringFromURLConfig{}
	}
	dcfg := cfg.DecodedConfig.(*setStringFromURLConfig)

	if err := dcfg.fillDefaults(); err != nil {
		return nil, fmt.Errorf("can't configure SetStringFromURL filter: %v", err)
	}

	f, ok := cfg.FieldByName(dcfg.Field)
	if !ok {
		return nil, fmt.Errorf("SetStringFromURL: unknow field %q", dcfg.Field)
	}

	strings := make([][]byte, 0, len(dcfg.Strings))
	for _, s := range dcfg.Strings {
		strings = append(strings, []byte(s))
	}

	return &setStringFromURL{field: f, strings: strings}, nil
}

func (f *setStringFromURL) Stats() baker.FilterStats {
	return baker.FilterStats{
		NumProcessedLines: atomic.LoadInt64(&f.numProcessedLines),
		NumFilteredLines:  atomic.LoadInt64(&f.numFilteredLines),
	}
}

func (f *setStringFromURL) Process(l baker.Record, next func(baker.Record)) {
	atomic.AddInt64(&f.numProcessedLines, 1)

	iurl, ok := l.Meta(inpututils.MetadataURL)
	if !ok {
		log.Infof("no metadata['url'] found on log line")
		atomic.AddInt64(&f.numFilteredLines, 1)
	}

	path := []byte(iurl.(*url.URL).Path)

	for _, s := range f.strings {
		if !bytes.Contains(path, s) {
			continue
		}

		l.Set(f.field, s)
		next(l)
	}

	atomic.AddInt64(&f.numFilteredLines, 1)
}
