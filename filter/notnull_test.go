package filter

import (
	"strings"
	"testing"

	"github.com/AdRoll/baker"
)

func TestNotNull(t *testing.T) {
	tests := []struct {
		record  string
		fields  []string
		want    bool // true: kept, false: discarded
		wantErr bool
	}{
		{
			record: "abc,def,ghi",
			fields: nil,
			want:   true,
		},
		{
			record:  "abc,def,ghi",
			fields:  []string{"foo", "non-existent"},
			wantErr: true,
		},
		{
			record: "abc,def,ghi",
			fields: []string{},
			want:   true,
		},
		{
			record: "abc,def,ghi",
			fields: []string{"foo"},
			want:   true,
		},
		{
			record: "abc,def,",
			fields: []string{"foo"},
			want:   true,
		},
		{
			record: "abc,def,",
			fields: []string{"foo", "bar", "baz"},
			want:   false,
		},
		{
			record: "abc,,ghi",
			fields: []string{"bar"},
			want:   false,
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
		t.Run(strings.Join(tt.fields, ","), func(t *testing.T) {
			f, err := NewNotNull(baker.FilterParams{
				ComponentParams: baker.ComponentParams{
					FieldByName: fieldByName,
					DecodedConfig: &NotNullConfig{
						Fields: tt.fields,
					},
				},
			})

			if (err != nil) != (tt.wantErr) {
				t.Fatalf("got error = %v, want error = %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			kept := false
			l := &baker.LogLine{FieldSeparator: ','}
			if err := l.Parse([]byte(tt.record), nil); err != nil {
				t.Fatalf("parse error: %q", err)
			}

			f.Process(l, func(baker.Record) { kept = true })

			if kept != tt.want {
				t.Errorf("got record kept=%t, want %t", kept, tt.want)
			}
		})
	}
}
