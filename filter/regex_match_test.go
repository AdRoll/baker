package filter

import (
	"fmt"
	"testing"

	"github.com/AdRoll/baker"
)

func TestRegexMatch(t *testing.T) {
	tests := []struct {
		record  string
		fields  []string
		regexs  []string
		want    bool // true: kept, false: discarded
		wantErr bool
	}{
		{
			fields:  []string{"foo"},
			regexs:  nil,
			wantErr: true,
		},
		{
			fields:  []string{"non-existent"},
			regexs:  []string{"foo"},
			wantErr: true,
		},
		{
			record: "abc,def,ghi",
			fields: []string{"foo"},
			regexs: []string{"^abc$"},
			want:   true,
		},
		{
			record: "abc,def,ghi",
			fields: []string{"foo"},
			regexs: []string{"^ab"},
			want:   true,
		},
		{
			record: "abc,def,ghi",
			fields: []string{"bar"},
			regexs: []string{"e"},
			want:   true,
		},
		{
			record: "abc,def,ghi",
			fields: []string{"foo"},
			regexs: []string{"^ab$"},
			want:   false,
		},
		{
			record: "abc,def,ghi",
			fields: []string{"foo", "baz"},
			regexs: []string{"^ab$", "ghi"},
			want:   false,
		},
		{
			record: "abc,def,ghi",
			fields: []string{"foo", "bar", "baz"},
			regexs: []string{"^ab$", ".*", `[a-z]{2}i`},
			want:   false,
		},
		{
			record: "abc,def,ghi",
			fields: []string{"foo", "bar", "baz"},
			regexs: []string{"^abc$", ".*", `[a-z]{2}i`},
			want:   true,
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
		t.Run(fmt.Sprintf("fields=%v regexs=%v", tt.fields, tt.regexs), func(t *testing.T) {
			f, err := NewRegexMatch(baker.FilterParams{
				ComponentParams: baker.ComponentParams{
					FieldByName: fieldByName,
					DecodedConfig: &RegexMatchConfig{
						Fields: tt.fields,
						Regexs: tt.regexs,
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
