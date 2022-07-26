package filter

import (
	"fmt"
	"testing"

	"github.com/AdRoll/baker"
)

func TestRegexMatch(t *testing.T) {
	tests := []struct {
		record        string
		fields        []string
		regexs        []string
		invert        bool
		wantErr       bool
		wantConfigErr bool
	}{
		{
			record:  "abc,def,ghi",
			fields:  []string{"field0"},
			regexs:  []string{"^abc$"},
			wantErr: false,
		},
		{
			record:  `goodstuff,def,ghi`,
			fields:  []string{"field0"},
			regexs:  []string{"good"},
			wantErr: false,
		},
		{
			record:  "abc,def,ghi",
			fields:  []string{"field0"},
			regexs:  []string{"^ab"},
			wantErr: false,
		},
		{
			record:  "abc,def,ghi",
			fields:  []string{"field1"},
			regexs:  []string{"e"},
			wantErr: false,
		},
		{
			record:  "abc,def,ghi",
			fields:  []string{"field0"},
			regexs:  []string{"^ab$"},
			wantErr: true,
		},
		{
			record:  "abc,def,ghi",
			fields:  []string{"field0", "field2"},
			regexs:  []string{"^ab$", "ghi"},
			wantErr: true,
		},
		{
			record:  "abc,def,ghi",
			fields:  []string{"field0", "field1", "field2"},
			regexs:  []string{"^ab$", ".*", `[a-z]{2}i`},
			wantErr: true,
		},
		{
			record:  "abc,def,ghi",
			fields:  []string{"field0", "field1", "field2"},
			regexs:  []string{"^abc$", ".*", `[a-z]{2}i`},
			wantErr: false,
		},

		// inverted
		{
			record:  `good,def,ghi`,
			fields:  []string{"field0"},
			regexs:  []string{"bad"},
			invert:  true,
			wantErr: false,
		},
		{
			record:  `["badstuff"],good,ghi`,
			fields:  []string{"field0", "field1"},
			regexs:  []string{"bad", "bad"},
			invert:  true,
			wantErr: true,
		},
		{
			record:  `["good"],bad,ghi`,
			fields:  []string{"field0", "field1"},
			regexs:  []string{"bad", "bad"},
			invert:  true,
			wantErr: true,
		},
		{
			record:  `good,good,ghi`,
			fields:  []string{"field0", "field1"},
			regexs:  []string{"bad", "bad"},
			invert:  true,
			wantErr: false,
		},

		// configuration errors
		{
			fields:        []string{"field0"},
			regexs:        nil,
			wantConfigErr: true,
		},
		{
			fields:        []string{"non-existent"},
			regexs:        []string{"field0"},
			wantConfigErr: true,
		},
	}

	fieldByName := func(name string) (baker.FieldIndex, bool) {
		switch name {
		case "field0":
			return 0, true
		case "field1":
			return 1, true
		case "field2":
			return 2, true
		}
		return 0, false
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("fields=%v/regexs=%v/invert=%t", tt.fields, tt.regexs, tt.invert), func(t *testing.T) {
			f, err := NewRegexMatch(baker.FilterParams{
				ComponentParams: baker.ComponentParams{
					FieldByName: fieldByName,
					DecodedConfig: &RegexMatchConfig{
						Fields:      tt.fields,
						Regexs:      tt.regexs,
						InvertMatch: tt.invert,
					},
				},
			})

			if (err != nil) != (tt.wantConfigErr) {
				t.Fatalf("got error = %s, want error = %t", err, tt.wantConfigErr)
			}

			if tt.wantConfigErr {
				return
			}

			l := &baker.LogLine{FieldSeparator: ','}
			if err := l.Parse([]byte(tt.record), nil); err != nil {
				t.Fatalf("parse error: %q", err)
			}

			if err := f.Process(l); (err != nil) != tt.wantErr {
				t.Errorf("Process returned err=%v want err=%v", err, tt.wantErr)
			}
		})
	}
}
