package baker

import (
	"bytes"
	"reflect"
	"testing"
	"time"
)

type dummyConfig struct {
	IntField              int               `help:"int field" required:"true" default:"0"`
	Int64Field            int64             `help:"int64 field" required:"false" default:"1"`
	DurationField         time.Duration     `help:"duration field" required:"true" default:"2s"`
	StringField           string            `help:"string field" required:"true" default:"4"`
	BoolField             bool              `help:"bool field" required:"true" default:"true"`
	SliceOfStringsField   []string          `help:"strings field" required:"true" default:"[\"a\", \"b\", \"c\"]"`
	SliceOfIntsField      []int             `help:"ints field" required:"true" default:"[0, 1, 2, 3]"`
	MapOfStringsToStrings map[string]string `help:"map of strings to strings field" required:"true" default:"{foo=\"bar\", bar=\"foo\"}"`
	MapOfStringsToInt     map[string]int    `help:"map of strings to ints field" required:"true" default:"{foo=12, bar=2}"`
}

var dummyKeys = []helpConfigKey{
	{
		name:     "IntField",
		typ:      "int",
		def:      "0",
		required: true,
		desc:     "int field",
	},
	{
		name:     "Int64Field",
		typ:      "int",
		def:      "1",
		required: false,
		desc:     "int64 field",
	},
	{
		name:     "DurationField",
		typ:      "duration",
		def:      "2s",
		required: true,
		desc:     "duration field",
	},
	{
		name:     "StringField",
		typ:      "string",
		def:      `"4"`,
		required: true,
		desc:     "string field",
	},
	{
		name:     "BoolField",
		typ:      "bool",
		def:      "true",
		required: true,
		desc:     "bool field",
	},
	{
		name:     "SliceOfStringsField",
		typ:      "array of strings",
		def:      `["a", "b", "c"]`,
		required: true,
		desc:     "strings field",
	},
	{
		name:     "SliceOfIntsField",
		typ:      "array of ints",
		def:      `[0, 1, 2, 3]`,
		required: true,
		desc:     "ints field",
	},
	{
		name:     "MapOfStringsToStrings",
		typ:      "map of strings to strings",
		def:      `{foo="bar", bar="foo"}`,
		required: true,
		desc:     "map of strings to strings field",
	},
	{
		name:     "MapOfStringsToInt",
		typ:      "map of strings to ints",
		def:      `{foo=12, bar=2}`,
		required: true,
		desc:     "map of strings to ints field",
	},
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
				IntField:              1,
				Int64Field:            2,
				DurationField:         3,
				StringField:           "5",
				BoolField:             false,
				SliceOfStringsField:   []string{"foo", "bar"},
				SliceOfIntsField:      []int{0, 1, 2, 3, 4, 5},
				MapOfStringsToStrings: map[string]string{"foo": "bar", "bar": "foo"},
				MapOfStringsToInt:     map[string]int{"foo": 12, "bar": 5},
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

func Test_newInputDoc(t *testing.T) {
	const (
		name = "dummy"
		help = "This is the high-level doc of the dummy input."
	)

	desc := InputDesc{
		Name:   name,
		Config: &dummyConfig{},
		Help:   help,
	}
	want := inputDoc{
		baseDoc: baseDoc{
			name: name,
			help: help,
			keys: dummyKeys,
		},
	}

	got, err := newInputDoc(desc)
	if err != nil {
		t.Errorf("newInputDoc() error = %v", err)
		return
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("newInputDoc():\ngot:\n%+v\nwant:\n%+v", got, want)
	}
}

func Test_newFilterDoc(t *testing.T) {
	const (
		name = "dummy"
		help = "This is the high-level doc of the dummy filter."
	)

	desc := FilterDesc{
		Name:   name,
		Config: &dummyConfig{},
		Help:   help,
	}
	want := filterDoc{
		baseDoc: baseDoc{
			name: name,
			help: help,
			keys: dummyKeys,
		},
	}

	got, err := newFilterDoc(desc)
	if err != nil {
		t.Errorf("newFilterDoc() error = %v", err)
		return
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("newFilterDoc():\ngot:\n%+v\nwant:\n%+v", got, want)
	}
}

func Test_newOuputDoc(t *testing.T) {
	const (
		name = "dummy"
		help = "This is the high-level doc of the dummy output."
	)

	desc := OutputDesc{
		Name:   name,
		Config: &dummyConfig{},
		Help:   help,
		Raw:    true,
	}
	want := outputDoc{
		raw: true,
		baseDoc: baseDoc{
			name: name,
			help: help,
			keys: dummyKeys,
		},
	}

	got, err := newOutputDoc(desc)
	if err != nil {
		t.Errorf("newOutputDoc() error = %v", err)
		return
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("newOutputDoc():\ngot:\n%+v\nwant:\n%+v", got, want)
	}
}

func Test_newUploadDoc(t *testing.T) {
	const (
		name = "dummy"
		help = "This is the high-level doc of the dummy upload."
	)

	desc := UploadDesc{
		Name:   name,
		Config: &dummyConfig{},
		Help:   help,
	}
	want := uploadDoc{
		baseDoc: baseDoc{
			name: name,
			help: help,
			keys: dummyKeys,
		},
	}

	got, err := newUploadDoc(desc)
	if err != nil {
		t.Errorf("newUploadDoc() error = %v", err)
		return
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("newUploadDoc():\ngot:\n%+v\nwant:\n%+v", got, want)
	}
}

func Test_newMetricsDoc(t *testing.T) {
	const (
		name = "dummy"
		help = "This is the high-level doc of the dummy metrics."
	)

	desc := MetricsDesc{
		Name:   name,
		Config: &dummyConfig{},
	}
	want := metricsDoc{
		name: name,
		keys: dummyKeys,
	}

	got, err := newMetricsDoc(desc)
	if err != nil {
		t.Errorf("newMetricsDoc() error = %v", err)
		return
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("newMetricsDoc():\ngot:\n%+v\nwant:\n%+v", got, want)
	}
}
