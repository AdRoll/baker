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
			fields:  []string{"foox"},
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

/*
func TestClearFieldsFilter(t *testing.T) {
	fieldByName := func(name string) (baker.FieldIndex, bool) {
		switch name {
		case "age":
			return 0, true
		case "name":
			return 1, true
		case "surname":
			return 2, true
		}
		return 0, false
	}

	params := baker.FilterParams{
		ComponentParams: baker.ComponentParams{
			FieldByName: fieldByName,
			DecodedConfig: &ClearFieldsConfig{
				Fields: []string{"name", "age", "surname"},
			},
		},
	}

	ll := baker.LogLine{FieldSeparator: ','}
	err := ll.Parse([]byte("12,Jim,Morrison,Paris,"), nil)
	if err != nil {
		t.Fatal(err)
	}

	f, err := NewClearFields(params)
	if err != nil {
		t.Fatal(err)
	}

	f.Process(&ll, func(baker.Record) {})

	for i := baker.FieldIndex(0); i < 3; i++ {
		if ll.Get(i) != nil {
			t.Errorf("field %d not cleared, got %q, want nil", i, ll.Get(i))
		}
	}

	if !bytes.Equal(ll.Get(3), []byte("Paris")) {
		t.Errorf("field 3 = %q, want %q", ll.Get(3), "Paris")
	}
}
*/
