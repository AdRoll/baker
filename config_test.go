package baker

import (
	"io/ioutil"
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
	buf, _ := ioutil.ReadAll(s)

	if want != string(buf) {
		t.Fatalf("wrong toml: %s", string(buf))
	}
}

func Test_assignFieldMapping(t *testing.T) {
	fieldName := func(f FieldIndex) string {
		switch f {
		case 0:
			return "name0"
		case 1:
			return "name1"
		}
		return ""
	}

	fieldByName := func(n string) (FieldIndex, bool) {
		switch n {
		case "name0":
			return 0, true
		case "name1":
			return 1, true
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
				FieldName:   fieldName,
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
				FieldName: fieldName,
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
				FieldName:   fieldName,
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
			comp: Components{
				FieldByName: fieldByName,
				FieldName:   fieldName,
			},
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
			if name := tt.cfg.fieldName(0); name != "name0" {
				t.Errorf(`cfg.fieldName(0) = %q, want %q`, name, "name0")
			}
			if name := tt.cfg.fieldName(1); name != "name1" {
				t.Errorf(`cfg.fieldName(1) = %q, want %q`, name, "name1")
			}
		})
	}
}
