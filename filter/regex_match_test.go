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
		invert  bool
		keep    bool // true: kept, false: discarded
		wantErr bool
	}{
		{
			record: "abc,def,ghi",
			fields: []string{"field0"},
			regexs: []string{"^abc$"},
			keep:   true,
		},
		{
			record: `goodstuff,def,ghi`,
			fields: []string{"field0"},
			regexs: []string{"good"},
			keep:   true,
		},
		{
			record: "abc,def,ghi",
			fields: []string{"field0"},
			regexs: []string{"^ab"},
			keep:   true,
		},
		{
			record: "abc,def,ghi",
			fields: []string{"field1"},
			regexs: []string{"e"},
			keep:   true,
		},
		{
			record: "abc,def,ghi",
			fields: []string{"field0"},
			regexs: []string{"^ab$"},
			keep:   false,
		},
		{
			record: "abc,def,ghi",
			fields: []string{"field0", "field2"},
			regexs: []string{"^ab$", "ghi"},
			keep:   false,
		},
		{
			record: "abc,def,ghi",
			fields: []string{"field0", "field1", "field2"},
			regexs: []string{"^ab$", ".*", `[a-z]{2}i`},
			keep:   false,
		},
		{
			record: "abc,def,ghi",
			fields: []string{"field0", "field1", "field2"},
			regexs: []string{"^abc$", ".*", `[a-z]{2}i`},
			keep:   true,
		},

		// inverted
		{
			record: `good,def,ghi`,
			fields: []string{"field0"},
			regexs: []string{"bad"},
			invert: true,
			keep:   true,
		},
		{
			record: `["badstuff"],good,ghi`,
			fields: []string{"field0", "field1"},
			regexs: []string{"bad", "bad"},
			invert: true,
			keep:   false,
		},
		{
			record: `["good"],bad,ghi`,
			fields: []string{"field0", "field1"},
			regexs: []string{"bad", "bad"},
			invert: true,
			keep:   false,
		},
		{
			record: `good,good,ghi`,
			fields: []string{"field0", "field1"},
			regexs: []string{"bad", "bad"},
			invert: true,
			keep:   true,
		},

		// error cases
		{
			fields:  []string{"field0"},
			regexs:  nil,
			wantErr: true,
		},
		{
			fields:  []string{"non-existent"},
			regexs:  []string{"field0"},
			wantErr: true,
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

			if kept != tt.keep {
				t.Errorf("got record kept=%t, want %t", kept, tt.keep)
			}
		})
	}
}
