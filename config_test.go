package baker

import (
	"io/ioutil"
	"os"
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
	src_toml := `
	[general]
	dont_validate_fields = ${DNT_VAL_FIELDS}
	`

	want_toml := `
	[general]
	dont_validate_fields = true
	`
	t.Run("no env var", func(t *testing.T) {
		_, err := replaceEnvVars(strings.NewReader(src_toml))
		if err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("with env var", func(t *testing.T) {
		os.Setenv("DNT_VAL_FIELDS", "true")
		defer os.Unsetenv("DNT_VAL_FIELDS")
		s, err := replaceEnvVars(strings.NewReader(src_toml))
		if err != nil {
			t.Fatalf("replaceEnvVars err: %v", err)
		}
		buf, _ := ioutil.ReadAll(s)

		if want_toml != string(buf) {
			t.Fatalf("wrong toml: %s", string(buf))
		}
	})
}

func TestEnvVarComplexReplace(t *testing.T) {
	src_toml := `
[input]
name = "InputName"
[input.config]
	some_key = [${BAKER_CONF_TEST_SLICE}] # strings slice
[[filter]]
name = "FilterName"
	[filter.config]
	some_key = ${BAKER_CONF_TEST_BOOL} # boolean
[output]
name = "OutputName"
procs = ${BAKER_CONF_TEST_NMBR} # number
	[output.config]
	some_key = "${BAKER_CONF_TEST_STR}" # string
	another_key = "${BAKER_CONF_TEST_STR}" # second occurrence of replacement
	`

	want_toml := `
[input]
name = "InputName"
[input.config]
	some_key = ["a", "b", "c"] # strings slice
[[filter]]
name = "FilterName"
	[filter.config]
	some_key = true # boolean
[output]
name = "OutputName"
procs = 7 # number
	[output.config]
	some_key = "a string value" # string
	another_key = "a string value" # second occurrence of replacement
	`
	os.Setenv("BAKER_CONF_TEST_SLICE", "\"a\", \"b\", \"c\"")
	os.Setenv("BAKER_CONF_TEST_BOOL", "true")
	os.Setenv("BAKER_CONF_TEST_NMBR", "7")
	os.Setenv("BAKER_CONF_TEST_STR", "a string value")
	defer func() {
		os.Unsetenv("BAKER_CONF_TEST_SLICE")
		os.Unsetenv("BAKER_CONF_TEST_BOOL")
		os.Unsetenv("BAKER_CONF_TEST_NMBR")
		os.Unsetenv("BAKER_CONF_TEST_STR")
	}()

	s, err := replaceEnvVars(strings.NewReader(src_toml))
	if err != nil {
		t.Fatalf("replaceEnvVars err: %v", err)
	}
	buf, _ := ioutil.ReadAll(s)

	if want_toml != string(buf) {
		t.Fatalf("wrong toml: %s", string(buf))
	}
}
