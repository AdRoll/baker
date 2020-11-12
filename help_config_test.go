package baker

import (
	"bytes"
	"testing"
	"time"
)

type dummyConfig struct {
	IntField            int           `help:"int field" required:"true" default:"0"`
	Int64Field          int64         `help:"int64 field" required:"false" default:"1"`
	DurationField       time.Duration `help:"duration field" required:"true" default:"2s"`
	StringField         string        `help:"string field" required:"true" default:"4"`
	BoolField           bool          `help:"bool field" required:"true" default:"true"`
	SliceOfStringsField []string      `help:"strings field" required:"true" default:"[\"a\", \"b\", \"c\"]"`
	SliceOfIntsField    []int         `help:"ints field" required:"true" default:"[0, 1, 2, 3]"`
}

func TestGenerateHelp(t *testing.T) {
	tests := []struct {
		name    string
		desc    interface{}
		wantErr bool
	}{
		{
			name:    "nil",
			desc:    nil,
			wantErr: true,
		},
		{
			name:    "unsupported type",
			desc:    23,
			wantErr: true,
		},
		{
			name: "supported configuration",
			desc: InputDesc{Name: "name", Config: &dummyConfig{
				IntField:            1,
				Int64Field:          2,
				DurationField:       3,
				StringField:         "5",
				BoolField:           false,
				SliceOfStringsField: []string{"foo", "bar"},
				SliceOfIntsField:    []int{0, 1, 2, 3, 4, 5},
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name+"/text", func(t *testing.T) {
			w := &bytes.Buffer{}
			if err := GenerateTextHelp(w, tt.desc); (err != nil) != tt.wantErr {
				t.Errorf("GenerateTextHelp() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && w.String() == "" {
				t.Errorf(`GenerateTextHelp() = "", shouldn't be empty`)
			}
		})
		t.Run(tt.name+"/markdown", func(t *testing.T) {
			w := &bytes.Buffer{}
			if err := GenerateMarkdownHelp(w, tt.desc); (err != nil) != tt.wantErr {
				t.Errorf("GenerateMarkdownHelp() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && w.String() == "" {
				t.Errorf(`GenerateMarkdownHelp() = "", shouldn't be empty`)
			}
		})
	}
}
