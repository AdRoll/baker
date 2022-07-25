package filter

import (
	"bytes"
	"testing"

	"github.com/AdRoll/baker"
)

func TestURLEscape(t *testing.T) {
	const fieldSeparator = '|'

	tests := []struct {
		name     string
		record   string
		src      string
		dst      string
		unescape bool
		want     []byte
		wantErr  bool
	}{
		{
			name:   "escape to another field",
			record: `{"this":"is", "an":["url", "encoded", "json", "string"]}` + "|" + "foo" + "|" + "bar",
			src:    "f0",
			dst:    "f1",
			want: []byte(`{"this":"is", "an":["url", "encoded", "json", "string"]}` +
				"|" + "%7B%22this%22%3A%22is%22%2C+%22an%22%3A%5B%22url%22%2C+%22encoded%22%2C+%22json%22%2C+%22string%22%5D%7D" +
				"|" + "bar",
			),
		},
		{
			name:   "escape to same field",
			record: "foo" + "|" + `{"this":"is", "an":["url", "encoded", "json", "string"]}` + "|" + "bar",
			src:    "f1",
			dst:    "f1",
			want: []byte("foo" +
				"|" + "%7B%22this%22%3A%22is%22%2C+%22an%22%3A%5B%22url%22%2C+%22encoded%22%2C+%22json%22%2C+%22string%22%5D%7D" +
				"|" + "bar",
			),
		},
		{
			name:     "unescape to another field",
			record:   "%7B%22this%22%3A%22is%22%2C+%22an%22%3A%5B%22url%22%2C+%22encoded%22%2C+%22json%22%2C+%22string%22%5D%7D" + "|" + "foo" + "|" + "bar",
			unescape: true,
			src:      "f0",
			dst:      "f1",
			want: []byte("%7B%22this%22%3A%22is%22%2C+%22an%22%3A%5B%22url%22%2C+%22encoded%22%2C+%22json%22%2C+%22string%22%5D%7D" +
				"|" + `{"this":"is", "an":["url", "encoded", "json", "string"]}` +
				"|" + "bar",
			),
		},
		{
			name:     "unescape to same field",
			record:   "foo" + "|" + "%7B%22this%22%3A%22is%22%2C+%22an%22%3A%5B%22url%22%2C+%22encoded%22%2C+%22json%22%2C+%22string%22%5D%7D" + "|" + "bar",
			unescape: true,
			src:      "f1",
			dst:      "f1",
			want: []byte("foo" +
				"|" + `{"this":"is", "an":["url", "encoded", "json", "string"]}` +
				"|" + "bar",
			),
		},
		{
			name:     "unescape error to another field",
			record:   "%7B%" + "|" + "foo" + "|" + "bar",
			unescape: true,
			src:      "f0",
			dst:      "f1",
			want: []byte("%7B%" +
				"|" + "" +
				"|" + "bar",
			),
		},
		{
			name:     "unescape error to same field",
			record:   "foo" + "|" + "%7B%" + "|" + "bar",
			unescape: true,
			src:      "f1",
			dst:      "f1",
			want: []byte("foo" +
				"|" + "" +
				"|" + "bar",
			),
		},
		// errors
		{
			name:    "SrcField doesn't exist",
			src:     "foo",
			dst:     "f0",
			wantErr: true,
		},
		{
			name:    "DstField doesn't exist",
			src:     "f0",
			dst:     "bar",
			wantErr: true,
		},
	}

	fieldByName := func(name string) (baker.FieldIndex, bool) {
		switch name {
		case "f0":
			return 0, true
		case "f1":
			return 1, true
		case "f2":
			return 2, true
		}
		return 0, false
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := NewURLEscape(baker.FilterParams{
				ComponentParams: baker.ComponentParams{
					FieldByName: fieldByName,
					DecodedConfig: &URLEscapeConfig{
						SrcField: tt.src,
						DstField: tt.dst,
						Unescape: tt.unescape,
					},
				},
			})

			if (err != nil) != (tt.wantErr) {
				t.Fatalf("got error = %v, want error = %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			rec := &baker.LogLine{FieldSeparator: fieldSeparator}
			if err := rec.Parse([]byte(tt.record), nil); err != nil {
				t.Fatalf("parse error: %q", err)
			}

			if err := f.Process(rec); err != nil {
				// Record untouched in case of error
				return
			}
			if !bytes.Equal(rec.ToText(nil), tt.want) {
				t.Errorf("got:\n%q\n\nwant:\n%q", rec.ToText(nil), tt.want)
			}
		})
	}
}
