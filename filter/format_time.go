package filter

import (
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/AdRoll/baker"
	log "github.com/sirupsen/logrus"
)

const (
	formatTimeHelp = `
This filter formats and converts time strings from one format to another. The filter requires the 
source and the destination field name along with the time format of the two fields. Most standard 
formats are supported, but it is possible to provide a custom one using layout string, 
i.e. [Go time layout](https://pkg.go.dev/time#pkg-constants).

Supported time format are:
- ` + "`ANSIC`" + ` format: "Mon Jan _2 15:04:05 2006"
- ` + "`UnixDate`" + ` format: "Mon Jan _2 15:04:05 MST 2006"
- ` + "`RubyDate`" + ` format: "Mon Jan 02 15:04:05 -0700 2006"
- ` + "`RFC822`" + ` format: "02 Jan 06 15:04 MST"
- ` + "`RFC822Z`" + ` that is RFC822 with numeric zone, format: "02 Jan 06 15:04 -0700"
- ` + "`RFC850`" + ` format: "Monday, 02-Jan-06 15:04:05 MST"
- ` + "`RFC1123`" + ` format: "Mon, 02 Jan 2006 15:04:05 MST"
- ` + "`RFC1123Z`" + ` that is RFC1123 with numeric zone, format: "Mon, 02 Jan 2006 15:04:05 -0700"
- ` + "`RFC3339`" + ` format: "2006-01-02T15:04:05Z07:00"
- ` + "`RFC3339Nano`" + ` format: "2006-01-02T15:04:05.999999999Z07:00"
- ` + "`unix`" + ` unix epoch in seconds
- ` + "`unixns`" + ` unix epoch in nanoseconds
`
	ansic       = "ANSIC"
	unixdate    = "UnixDate"
	rubydate    = "RubyDate"
	rfc822      = "RFC822"
	rfc822z     = "RFC822Z"
	rfc850      = "RFC850"
	rfc1123     = "RFC1123"
	rfc1123z    = "RFC1123Z"
	rfc3339     = "RFC3339"
	rfc3339nano = "RFC3339Nano"
	unix        = "unix"
	unixns      = "unixns"
)

var FormatTimeDesc = baker.FilterDesc{
	Name:   "FormatTime",
	New:    NewFormatTime,
	Config: &FormatTimeConfig{},
	Help:   formatTimeHelp,
}

type FormatTimeConfig struct {
	SrcField  string `help:"Field name of the input time" required:"true"`
	DstField  string `help:"Field name of the output time" required:"true"`
	SrcFormat string `help:"Format of the input time" required:"false" default:"UnixDate"`
	DstFormat string `help:"Format of the output time" required:"false" default:"unix"`
}

func (cfg *FormatTimeConfig) fillDefaults() {
	if cfg.SrcFormat == "" {
		cfg.SrcFormat = unixdate
	}
	if cfg.DstFormat == "" {
		cfg.DstFormat = unix
	}
}

type FormatTime struct {
	cfg *FormatTimeConfig

	src    baker.FieldIndex
	dst    baker.FieldIndex
	parse  func(t []byte) (time.Time, error)
	format func(t time.Time) []byte

	// Shared state
	numProcessedLines int64
	numFilteredLines  int64
}

func NewFormatTime(cfg baker.FilterParams) (baker.Filter, error) {
	dcfg := cfg.DecodedConfig.(*FormatTimeConfig)
	dcfg.fillDefaults()

	f := &FormatTime{cfg: dcfg}

	idx, ok := cfg.FieldByName(dcfg.SrcField)
	if !ok {
		return nil, fmt.Errorf("unknown field %q", dcfg.SrcField)
	}
	f.src = idx

	switch dcfg.SrcFormat {
	case unix:
		f.parse = func(b []byte) (time.Time, error) {
			sec, err := strconv.ParseInt(string(b), 10, 64)
			if err != nil {
				return time.Time{}, err
			}
			return time.Unix(sec, 0), nil
		}
	case unixns:
		f.parse = func(b []byte) (time.Time, error) {
			nsec, err := strconv.ParseInt(string(b), 10, 64)
			if err != nil {
				return time.Time{}, err
			}
			return time.Unix(0, nsec), nil
		}
	default:
		layout := formatToLayout(dcfg.SrcFormat)
		f.parse = func(b []byte) (time.Time, error) {
			t, err := time.Parse(layout, string(b))
			if err != nil {
				return time.Time{}, err
			}
			return t, nil
		}
	}

	idx, ok = cfg.FieldByName(dcfg.DstField)
	if !ok {
		return nil, fmt.Errorf("unknown field %q", dcfg.DstField)
	}
	f.dst = idx

	switch dcfg.DstFormat {
	case unix:
		f.format = func(t time.Time) []byte {
			return strconv.AppendInt(nil, t.Unix(), 10)
		}
	case unixns:
		f.format = func(t time.Time) []byte {
			return strconv.AppendInt(nil, t.UnixNano(), 10)
		}
	default:
		layout := formatToLayout(dcfg.DstFormat)
		f.format = func(t time.Time) []byte {
			return []byte(t.Format(layout))
		}
	}

	return f, nil
}

func (f *FormatTime) Stats() baker.FilterStats {
	return baker.FilterStats{
		NumProcessedLines: atomic.LoadInt64(&f.numProcessedLines),
		NumFilteredLines:  atomic.LoadInt64(&f.numFilteredLines),
	}
}

func (f *FormatTime) Process(l baker.Record, next func(baker.Record)) {
	t, err := f.parse(l.Get(f.src))
	if err != nil {
		log.Errorf("can't parse time: %v", err)
		atomic.AddInt64(&f.numFilteredLines, 1)
		return
	}

	l.Set(f.dst, f.format(t))
	atomic.AddInt64(&f.numProcessedLines, 1)
	next(l)
}

func formatToLayout(format string) string {
	switch format {
	case ansic:
		return time.ANSIC
	case unixdate:
		return time.UnixDate
	case rubydate:
		return time.RubyDate
	case rfc822:
		return time.RFC822
	case rfc822z:
		return time.RFC822Z
	case rfc850:
		return time.RFC850
	case rfc1123:
		return time.RFC1123
	case rfc1123z:
		return time.RFC1123Z
	case rfc3339:
		return time.RFC3339
	case rfc3339nano:
		return time.RFC3339Nano
	default:
		return format
	}
}
