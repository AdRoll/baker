package filter

import (
	"fmt"
	"testing"

	"github.com/AdRoll/baker"
)

func TestStringMatch(t *testing.T) {
	tests := []struct {
		field   string // we use this to both represent the name of the field as well as its value for simplicity
		strings []string
		invert  bool
		want    bool // true: kept, false: discarded
		wantErr bool
	}{
		{
			field:   "foo",
			strings: []string{},
			wantErr: true,
		},
		{
			field:   "foo",
			strings: []string{"foo"},
			want:    false,
		},
		{
			field:   "foo",
			strings: []string{"foo", "bar"},
			want:    false,
		},
		{
			field:   "foo",
			strings: []string{"fox", "baz"},
			want:    true,
		},
		{
			field:   "foo",
			strings: []string{"foo"},
			invert:  true,
			want:    true,
		},
		{
			field:   "foo",
			strings: []string{"foo", "bar"},
			invert:  true,
			want:    true,
		},
		{
			field:   "foo",
			strings: []string{"fox", "baz"},
			invert:  true,
			want:    false,
		},
	}

	fieldByName := func(name string) (baker.FieldIndex, bool) {
		switch name {
		case "foo":
			return 0, true
		case "bar":
			return 1, true
		case "baz":
			return 2, true
		}
		return 0, false
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("field=%v strings=%v invert=%t", tt.field, tt.strings, tt.invert), func(t *testing.T) {
			f, err := NewStringMatch(baker.FilterParams{
				ComponentParams: baker.ComponentParams{
					FieldByName: fieldByName,
					DecodedConfig: &StringMatchConfig{
						Field:   tt.field,
						Strings: tt.strings,
						Invert:  tt.invert,
					},
				},
			})

			if (err != nil) != (tt.wantErr) {
				t.Fatalf("got error = %s, want error = %t", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			kept := false

			fidx, ok := fieldByName(tt.field)
			if !ok {
				panic("wrong field should have triggered an error above")
			}

			var l baker.LogLine
			l.Set(fidx, []byte(tt.field))

			f.Process(&l, func(baker.Record) { kept = true })

			if kept != tt.want {
				t.Errorf("got record kept=%t, want %t", kept, tt.want)
			}
		})
	}
}
