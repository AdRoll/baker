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
			name: "unixms->" + fout,
			in:   fmt.Sprintf("%d", refTime.UnixNano()/int64(time.Millisecond)),
			fin:  "unixms",
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
			name: fin + "->unixms",
			in:   refTime.Format(lin),
			fin:  fin,
			out:  fmt.Sprintf("%d", timeAsFrom(lin, refTime).UnixNano()/int64(time.Millisecond)),
			fout: "unixms",
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
		name   string
		record string

		srcField  string
		srcFormat string
		dstField  string
		dstFormat string

		want    string
		wantErr bool // error during filter instantiation
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
			dstFormat: "", // unixms
			want:      "932183424000",
		},

		//errors
		{
			name:     "SrcField error",
			srcField: "not-exist",
			dstField: "f2",
			wantErr:  true,
		},
		{
			name:     "DstField error",
			srcField: "f2",
			dstField: "not-exist",
			wantErr:  true,
		},
		{
			name:      "format error",
			record:    "Sat Jul 17 03:50:24 UTC 1999,not-empty",
			srcField:  "f1",
			srcFormat: "foo bar",
			dstField:  "f2",
			want:      "", // dst field shoul empty on error
		},
		{
			name:      "unix time error",
			record:    "foobar,not-empty",
			srcField:  "f1",
			srcFormat: "unix",
			dstField:  "f2",
			want:      "", // dst field shoul empty on error
		},
		{
			name:      "unixms time error",
			record:    "foobar,not-empty",
			srcField:  "f1",
			srcFormat: "unixms",
			dstField:  "f2",
			want:      "", // dst field shoul empty on error
		},
		{
			name:      "unixns time error",
			record:    "foobar,not-empty",
			srcField:  "f1",
			srcFormat: "unixns",
			dstField:  "f2",
			want:      "", // dst field shoul empty on error
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

			if (err != nil) != (tt.wantErr) {
				t.Fatalf("got error = %v, want error = %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			rec1 := &baker.LogLine{FieldSeparator: ','}
			if err := rec1.Parse([]byte(tt.record), nil); err != nil {
				t.Fatalf("parse error: %q", err)
			}

			f.Process(rec1, func(rec2 baker.Record) {
				id, ok := fieldByName(tt.dstField)
				if !ok {
					t.Fatalf("cannot find field name")
				}
				time := rec2.Get(id)
				if string(time) != tt.want {
					t.Errorf("got time %q, want %q", time, tt.want)
				}
			})
		})
	}
}
