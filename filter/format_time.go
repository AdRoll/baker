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
This filter formats and converts date/time strings from one format to another. 
It requires the source and destination field names along with 2 format strings, the 
first one indicates how to parse the input field while the second how to format it.

The source time parsing can fail if the time value does not match the provided format.
In this situation the filter clears the destination field, thus the user can filter out 
those results with a __NotNull__ filter.

Most standard formats are supported out of the box and you can provide your own format 
string, see [Go time layout](https://pkg.go.dev/time#pkg-constants).

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
- ` + "`unixms`" + ` unix epoch in milliseconds
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
	unixms      = "unixms"
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
	DstFormat string `help:"Format of the output time" required:"false" default:"unixms"`
}

func (cfg *FormatTimeConfig) fillDefaults() {
	if cfg.SrcFormat == "" {
		cfg.SrcFormat = unixdate
	}
	if cfg.DstFormat == "" {
		cfg.DstFormat = unixms
	}
}

type FormatTime struct {
	src    baker.FieldIndex
	dst    baker.FieldIndex
	parse  func(t []byte) (time.Time, error)
	format func(t time.Time) []byte

	// Shared state
	numProcessedLines int64
}

func NewFormatTime(cfg baker.FilterParams) (baker.Filter, error) {
	dcfg := cfg.DecodedConfig.(*FormatTimeConfig)
	dcfg.fillDefaults()

	f := &FormatTime{}

	idx, ok := cfg.FieldByName(dcfg.SrcField)
	if !ok {
		return nil, fmt.Errorf("unknown field %q", dcfg.SrcField)
	}
	f.src = idx

	idx, ok = cfg.FieldByName(dcfg.DstField)
	if !ok {
		return nil, fmt.Errorf("unknown field %q", dcfg.DstField)
	}
	f.dst = idx

	f.parse = genParseFun(dcfg.SrcFormat)
	f.format = genFormatFun(dcfg.DstFormat)

	return f, nil
}

func (f *FormatTime) Stats() baker.FilterStats {
	return baker.FilterStats{
		NumProcessedLines: atomic.LoadInt64(&f.numProcessedLines),
	}
}

func (f *FormatTime) Process(l baker.Record, next func(baker.Record)) {
	atomic.AddInt64(&f.numProcessedLines, 1)

	t, err := f.parse(l.Get(f.src))
	if err != nil {
		log.Errorf("can't parse time: %v", err)
		l.Set(f.dst, nil)
	} else {
		l.Set(f.dst, f.format(t))
	}

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

func genParseFun(format string) func(b []byte) (time.Time, error) {
	switch format {
	case unix:
		return func(b []byte) (time.Time, error) {
			sec, err := strconv.ParseInt(string(b), 10, 64)
			if err != nil {
				return time.Time{}, err
			}
			return time.Unix(sec, 0), nil
		}
	case unixms:
		return func(b []byte) (time.Time, error) {
			msec, err := strconv.ParseInt(string(b), 10, 64)
			if err != nil {
				return time.Time{}, err
			}
			return time.Unix(0, msec*int64(time.Millisecond)), nil
		}
	case unixns:
		return func(b []byte) (time.Time, error) {
			nsec, err := strconv.ParseInt(string(b), 10, 64)
			if err != nil {
				return time.Time{}, err
			}
			return time.Unix(0, nsec), nil
		}
	default:
		layout := formatToLayout(format)
		return func(b []byte) (time.Time, error) {
			t, err := time.Parse(layout, string(b))
			if err != nil {
				return time.Time{}, err
			}
			return t, nil
		}
	}
}

func genFormatFun(format string) func(t time.Time) []byte {
	switch format {
	case unix:
		return func(t time.Time) []byte {
			return []byte(strconv.FormatInt(t.Unix(), 10))
		}
	case unixms:
		return func(t time.Time) []byte {
			return []byte(strconv.FormatInt(t.UnixNano()/int64(time.Millisecond), 10))
		}
	case unixns:
		return func(t time.Time) []byte {
			return []byte(strconv.FormatInt(t.UnixNano(), 10))
		}
	default:
		layout := formatToLayout(format)
		return func(t time.Time) []byte {
			return []byte(t.Format(layout))
		}
	}
}
