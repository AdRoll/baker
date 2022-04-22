package baker

import (
	"io"
	"strings"
	"testing"
)

func TestFillCreateRecordDefault(t *testing.T) {
	tests := []struct {
		name    string
		field   string
		want    byte
		wantErr bool
	}{
		{
			name:  "default",
			field: "",
			want:  DefaultLogLineFieldSeparator,
		},
		{
			name:  "explicit comma",
			field: ",",
			want:  DefaultLogLineFieldSeparator,
		},
		{
			name:  "record separator",
			field: "\u001e",
			want:  0x1e,
		},
		{
			name:  "dot",
			field: ".",
			want:  '.',
		},
		{
			name:    "not ascii",
			field:   "à",
			wantErr: true,
		},
		{
			name:    "2 chars",
			field:   ",,",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				CSV: ConfigCSV{
					FieldSeparator: tt.field,
				},
			}
			err := cfg.fillCreateRecordDefault()
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Config.fillCreateRecordDefault() err: %v, wantErr: %v", err, tt.wantErr)
				}
				return
			}

			if sep := cfg.createRecord().(*LogLine).FieldSeparator; sep != tt.want {
				t.Errorf(`got separator "%c" (%v), want "%c" (%v)`, sep, sep, tt.want, tt.want)
			}
		})
	}
}

func TestEnvVarBaseReplace(t *testing.T) {
	src := `
	[general]
	dont_validate_fields = ${DNT_VAL_FIELDS}
	alt_form = "$ALT_FORM"
	unexisting_var = "${THIS_DOESNT_EXIST}"
	`

	want := `
	[general]
	dont_validate_fields = true
	alt_form = "ok"
	unexisting_var = ""
	`

	mapper := func(v string) string {
		switch v {
		case "DNT_VAL_FIELDS":
			return "true"
		case "ALT_FORM":
			return "ok"
		}
		return ""
	}

	s, err := replaceEnvVars(strings.NewReader(src), mapper)
	if err != nil {
		t.Fatalf("replaceEnvVars err: %v", err)
	}
	buf, _ := io.ReadAll(s)

	if want != string(buf) {
		t.Fatalf("wrong toml: %s", string(buf))
	}
}

func Test_assignFieldMapping(t *testing.T) {
	fieldNames := []string{"name0", "name1"}
	fieldByName := func(n string) (FieldIndex, bool) {
		for idx, name := range fieldNames {
			if n == name {
				return FieldIndex(idx), true
			}
		}

		return 0, false
	}

	cfgFields := ConfigFields{
		Names: []string{"name0", "name1"},
	}

	tests := []struct {
		name    string
		cfg     *Config
		comp    Components
		wantErr bool
	}{
		{
			name: "only in Config",
			cfg: &Config{
				Fields: cfgFields,
			},
		},
		{
			name: "only in Components",
			cfg:  &Config{},
			comp: Components{
				FieldByName: fieldByName,
				FieldNames:  fieldNames,
			},
		},

		// error cases
		{
			name: "nothing set",
			cfg:  &Config{}, comp: Components{},
			wantErr: true,
		},
		{
			name: "FieldByName but not FieldName",
			cfg:  &Config{},
			comp: Components{
				FieldByName: fieldByName,
			},
			wantErr: true,
		},
		{
			name: "FieldName but not FieldByName",
			cfg:  &Config{},
			comp: Components{
				FieldNames: fieldNames,
			},
			wantErr: true,
		},
		{
			name: "mapping set both in Config and Components",
			cfg: &Config{
				Fields: cfgFields,
			},
			comp: Components{
				FieldByName: fieldByName,
				FieldNames:  fieldNames,
			},
			wantErr: true,
		},
		{
			name: "duplicated field name",
			cfg: &Config{
				Fields: ConfigFields{
					Names: []string{"foo", "bar", "baz", "baz"},
				},
			},
			comp:    Components{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := assignFieldMapping(tt.cfg, tt.comp); (err != nil) != tt.wantErr {
				t.Errorf("assignFieldMapping() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			// Now we check that fieldByName has been set correctly.
			if field, ok := tt.cfg.fieldByName("name0"); field != 0 || !ok {
				t.Errorf(`cfg.fieldByName("name0") = %v,%v, want %v,%v`, field, ok, 0, true)
			}
			if field, ok := tt.cfg.fieldByName("name1"); field != 1 || !ok {
				t.Errorf(`cfg.fieldByName("name1") = %v,%v, want %v,%v`, field, ok, 1, true)
			}
			if field, ok := tt.cfg.fieldByName("do-no-exist"); ok {
				t.Errorf(`cfg.fieldByName("do-no-exist") = %v,%v, want %v,%v`, field, ok, 0, false)
			}

			// Now we check that fieldName has been set correctly.
			if name := tt.cfg.fieldNames[0]; name != "name0" {
				t.Errorf(`cfg.fieldNames[0] = %q, want %q`, name, "name0")
			}
			if name := tt.cfg.fieldNames[1]; name != "name1" {
				t.Errorf(`cfg.fieldNames[1] = %q, want %q`, name, "name1")
			}
		})
	}
}

type dummyRecord map[FieldIndex]string

func (r dummyRecord) Parse([]byte, Metadata) error {
	return nil
}
func (r dummyRecord) ToText(buf []byte) []byte {
	return []byte("")
}
func (r dummyRecord) Copy() Record {
	return Record(r.Copy())
}
func (r dummyRecord) Clear() {
	r.Clear()
}
func (r dummyRecord) Get(i FieldIndex) []byte {
	v, ok := r[i]
	if !ok {
		return make([]byte, 0)
	}
	return []byte(v)
}
func (r dummyRecord) Set(i FieldIndex, b []byte) {
	r[i] = string(b)
}
func (r dummyRecord) Meta(key string) (v interface{}, ok bool) {
	return r, true
}
func (r dummyRecord) Cache() *Cache {
	return &Cache{}
}

func Test_assignValidationMapping(t *testing.T) {
	fieldNames := []string{"name0", "name1"}
	fieldByName := func(n string) (FieldIndex, bool) {
		for idx, name := range fieldNames {
			if n == name {
				return FieldIndex(idx), true
			}
		}

		return 0, false
	}

	cfgValidation := ConfigValidation{"name0": "^val$", "name1": "^val$"}
	validate := func(r Record) (bool, FieldIndex) {
		if "val" != string(r.Get(0)) {
			return false, 0
		}
		if "val" != string(r.Get(1)) {
			return false, 1
		}
		return true, 0
	}

	tests := []struct {
		name      string
		cfg       *Config
		comp      Components
		wantErr   bool
		skipCheck bool
	}{
		{
			name: "only in Config",
			cfg: &Config{
				Validation:  cfgValidation,
				fieldByName: fieldByName, // needed by func assignValidationMapping
			},
			comp: Components{},
		},
		{
			name: "only in Components",
			cfg: &Config{
				fieldByName: fieldByName, // needed by func assignValidationMapping
			},
			comp: Components{
				Validate: validate,
			},
		},
		{
			name:      "nothing set",
			cfg:       &Config{},
			comp:      Components{},
			skipCheck: true, // check only that cfg.validate is not nil
		},

		// error cases
		{
			name: "validation set both in Config and Components",
			cfg: &Config{
				Validation:  cfgValidation,
				fieldByName: fieldByName, // needed by func assignValidationMapping
			},
			comp: Components{
				Validate: validate,
			},
			wantErr: true,
		},
		{
			name: "not existing field name",
			cfg: &Config{
				Validation:  ConfigValidation{"badname": "^val$"},
				fieldByName: fieldByName, // needed by func assignValidationMapping
			},
			comp:    Components{},
			wantErr: true,
		},
		{
			name: "validation regex not compile",
			cfg: &Config{
				Validation:  ConfigValidation{"name0": "*"},
				fieldByName: fieldByName, // needed by func assignValidationMapping
			},
			comp:    Components{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := assignValidationMapping(tt.cfg, tt.comp); (err != nil) != tt.wantErr {
				t.Errorf("assignValidationMapping() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}

			if tt.cfg.validate == nil {
				t.Errorf("cfg.validate should always be set")
			}
			if tt.skipCheck {
				return
			}

			// Now we check that validate has been set correctly.
			rec := dummyRecord{0: "val", 1: "val"}
			if ok, field := tt.cfg.validate(rec); !ok || field != 0 {
				t.Errorf(`cfg.validate("%v") = %v,%v, want %v,%v`, rec, ok, field, true, 0)
			}
			rec = dummyRecord{0: "badval", 1: "val"}
			if ok, field := tt.cfg.validate(rec); ok && field != 0 {
				t.Errorf(`cfg.validate("%v") = %v,%v, want %v,%v`, rec, ok, field, false, 0)
			}
			rec = dummyRecord{0: "val", 1: "badval"}
			if ok, field := tt.cfg.validate(rec); ok && field != 1 {
				t.Errorf(`cfg.validate("%v") = %v,%v, want %v,%v`, rec, ok, field, false, 1)
			}
		})
	}
}
