package filter

import (
	"strings"
	"testing"

	"github.com/AdRoll/baker"
)

func TestNotNull(t *testing.T) {
	tests := []struct {
		record        string
		fields        []string
		wantErr       bool
		wantConfigErr bool
	}{
		{
			record:  "abc,def,ghi",
			fields:  nil,
			wantErr: false,
		},
		{
			record:        "abc,def,ghi",
			fields:        []string{"foo", "non-existent"},
			wantConfigErr: true,
		},
		{
			record:  "abc,def,ghi",
			fields:  []string{},
			wantErr: false,
		},
		{
			record:  "abc,def,ghi",
			fields:  []string{"foo"},
			wantErr: false,
		},
		{
			record:  "abc,def,",
			fields:  []string{"foo"},
			wantErr: false,
		},
		{
			record:  "abc,def,",
			fields:  []string{"foo", "bar", "baz"},
			wantErr: true,
		},
		{
			record:  "abc,,ghi",
			fields:  []string{"bar"},
			wantErr: true,
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

			if (err != nil) != (tt.wantConfigErr) {
				t.Fatalf("got error = %v, want error = %v", err, tt.wantConfigErr)
			}

			if tt.wantConfigErr {
				return
			}

			l := &baker.LogLine{FieldSeparator: ','}
			if err := l.Parse([]byte(tt.record), nil); err != nil {
				t.Fatalf("parse error: %q", err)
			}

			err = f.Process(l)
			if (err != nil) != tt.wantErr {
				t.Errorf("Process() = %v, want err=%v", err, tt.wantErr)
			}
		})
	}
}
