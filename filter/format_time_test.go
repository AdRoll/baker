package filter

import (
	"fmt"
	"testing"
	"time"

	"github.com/AdRoll/baker"
)

func TestFormatTime(t *testing.T) {
	timeAsFrom := func(layout string, t time.Time) time.Time {
		out, _ := time.Parse(layout, t.Format(layout))
		return out
	}

	format := map[string]string{
		"ANSIC":       time.ANSIC,
		"UnixDate":    time.UnixDate,
		"RubyDate":    time.RubyDate,
		"RFC822":      time.RFC822,
		"RFC822Z":     time.RFC822Z,
		"RFC850":      time.RFC850,
		"RFC1123":     time.RFC1123,
		"RFC1123Z":    time.RFC1123Z,
		"RFC3339":     time.RFC3339,
		"RFC3339Nano": time.RFC3339Nano,
	}
	refTime := time.Unix(932183424, 0)

	type test struct {
		name string
		in   string
		fin  string
		out  string
		fout string
	}
	tests := []test{}

	for fin, lin := range format {
		for fout, lout := range format {
			t := test{
				name: fin + "->" + fout,
				in:   refTime.Format(lin),
				fin:  fin,
				out:  timeAsFrom(lin, refTime).Format(lout),
				fout: fout,
			}
			tests = append(tests, t)
		}
	}

	for fout, lout := range format {
		t := test{
			name: "unix->" + fout,
			in:   fmt.Sprintf("%d", refTime.Unix()),
			fin:  "unix",
			out:  refTime.Format(lout),
			fout: fout,
		}
		tests = append(tests, t)
	}
	for fout, lout := range format {
		t := test{
			name: "unixns->" + fout,
			in:   fmt.Sprintf("%d", refTime.UnixNano()),
			fin:  "unixns",
			out:  refTime.Format(lout),
			fout: fout,
		}
		tests = append(tests, t)
	}

	for fin, lin := range format {
		t := test{
			name: fin + "->unix",
			in:   refTime.Format(lin),
			fin:  fin,
			out:  fmt.Sprintf("%d", timeAsFrom(lin, refTime).Unix()),
			fout: "unix",
		}
		tests = append(tests, t)
	}
	for fin, lin := range format {
		t := test{
			name: fin + "->unixns",
			in:   refTime.Format(lin),
			fin:  fin,
			out:  fmt.Sprintf("%d", timeAsFrom(lin, refTime).UnixNano()),
			fout: "unixns",
		}
		tests = append(tests, t)
	}

	fieldByName := func(name string) (baker.FieldIndex, bool) {
		switch name {
		case "f1":
			return 0, true
		case "f2":
			return 1, true
		}
		return 0, false
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := NewFormatTime(baker.FilterParams{
				ComponentParams: baker.ComponentParams{
					FieldByName: fieldByName,
					DecodedConfig: &FormatTimeConfig{
						SrcField:  "f1",
						SrcFormat: tt.fin,
						DstField:  "f2",
						DstFormat: tt.fout,
					},
				},
			})

			if err != nil {
				t.Fatalf("got error = %v, want nil", err)
			}

			rec1 := &baker.LogLine{FieldSeparator: ';'}
			if err := rec1.Parse([]byte(tt.in), nil); err != nil {
				t.Fatalf("parse error: %q", err)
			}

			f.Process(rec1, func(rec2 baker.Record) {
				time := string(rec2.Get(1))
				if time != tt.out {
					t.Errorf("got time %q, want %q", time, tt.out)
				}
			})
		})
	}
}

func TestFormatTimeErrors(t *testing.T) {
	tests := []struct {
		name      string
		record    string
		srcField  string
		srcFormat string
		dstField  string
		dstFormat string
		want      string
		newErr    bool // error during filter instantiation
		procErr   bool // error during filter processing
	}{
		{
			name:      "custom format src",
			record:    "Jul-17-1999_03:50:24,",
			srcField:  "f1",
			srcFormat: "Jan-2-2006_15:04:05",
			dstField:  "f2",
			dstFormat: "UnixDate",
			want:      "Sat Jul 17 03:50:24 UTC 1999",
		},
		{
			name:      "custom format dst",
			record:    "Sat Jul 17 03:50:24 UTC 1999,",
			srcField:  "f1",
			srcFormat: "UnixDate",
			dstField:  "f2",
			dstFormat: "Jan-2-2006_15:04:05",
			want:      "Jul-17-1999_03:50:24",
		},
		{
			name:      "default format",
			record:    "Sat Jul 17 03:50:24 UTC 1999,",
			srcField:  "f1",
			srcFormat: "", // UnixDate
			dstField:  "f2",
			dstFormat: "", // unix
			want:      "932183424",
		},

		//errors
		{
			name:     "SrcField error",
			srcField: "not-exist",
			dstField: "f2",
			newErr:   true,
		},
		{
			name:     "DstField error",
			srcField: "f2",
			dstField: "not-exist",
			newErr:   true,
		},
		{
			name:      "format error",
			record:    "Sat Jul 17 03:50:24 UTC 1999,",
			srcField:  "f1",
			srcFormat: "foo bar",
			dstField:  "f2",
			procErr:   true,
		},
		{
			name:      "unix time error",
			record:    "foobar,",
			srcField:  "f1",
			srcFormat: "unix",
			dstField:  "f2",
			procErr:   true,
		},
		{
			name:      "unixns time error",
			record:    "foobar,",
			srcField:  "f1",
			srcFormat: "unixns",
			dstField:  "f2",
			procErr:   true,
		},
	}
	fieldByName := func(name string) (baker.FieldIndex, bool) {
		switch name {
		case "f1":
			return 0, true
		case "f2":
			return 1, true
		}
		return 0, false
	}
	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			f, err := NewFormatTime(baker.FilterParams{
				ComponentParams: baker.ComponentParams{
					FieldByName: fieldByName,
					DecodedConfig: &FormatTimeConfig{
						SrcField:  tt.srcField,
						SrcFormat: tt.srcFormat,
						DstField:  tt.dstField,
						DstFormat: tt.dstFormat,
					},
				},
			})

			if (err != nil) != (tt.newErr) {
				t.Fatalf("got error = %v, want error = %v", err, tt.newErr)
			}

			if tt.newErr {
				return
			}

			rec1 := &baker.LogLine{FieldSeparator: ','}
			if err := rec1.Parse([]byte(tt.record), nil); err != nil {
				t.Fatalf("parse error: %q", err)
			}

			callNext := false
			f.Process(rec1, func(rec2 baker.Record) {
				callNext = true
				id, ok := fieldByName(tt.dstField)
				if !ok {
					t.Fatalf("cannot find field name")
				}
				time := rec2.Get(id)
				if string(time) != tt.want {
					t.Errorf("got hash %q, want %q", time, tt.want)
				}
			})
			if !callNext && !tt.procErr {
				t.Fatalf("process error, want nil")
			}
		})
	}
}
