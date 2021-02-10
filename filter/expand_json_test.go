package filter

import (
	"bytes"
	"testing"

	"github.com/AdRoll/baker"
)

func TestExpandJSON(t *testing.T) {
	tests := []struct {
		name      string
		json      string
		field     map[string]string
		source    string
		trueFalse []string
		want      map[string]string
		wantErr   bool
	}{
		{
			name: "string test1",
			json: `{"j1": "value1", "j2": "value 2", "j3": "1,[2],3,4{"}`,
			want: map[string]string{"f1": "value1", "f2": "value 2", "f3": "1,[2],3,4{"},
		},
		{
			name: "string test2",
			json: `{"junk1": 234, "junk2": {"a":"hello","b":[true,2]}, "j1": "text"}`,
			want: map[string]string{"f1": "text"},
		},
		{
			name: "test numbers",
			json: `{"j1": 12, "j2": 0.15, "j3": 15.12312312312}`,
			want: map[string]string{"f1": "12", "f2": "0.15", "f3": "15.12312312312"},
		},
		{
			name: "test boolean 1",
			json: `{"j1": true, "j2": false}`,
			want: map[string]string{"f1": "true", "f2": "false"},
		},
		{
			name:      "test boolean 2",
			json:      `{"j1": true, "j2": false}`,
			want:      map[string]string{"f1": "t", "f2": "f"},
			trueFalse: []string{"t", "f"},
		},
		{
			name: "skip other json types",
			json: `{"j1": null, "j2": [1,2,3,4], "j3": {"j1":"a", "j2":false}}`,
			want: map[string]string{"f1": "", "f2": "", "f3": ""},
		},
		{
			name: "empty json",
			json: ``,
			want: map[string]string{"f1": "", "f2": ""},
		},
		{
			name: "json parse error",
			json: `{this is not a json]`,
			want: map[string]string{"f1": "", "f2": ""},
		},

		// errors
		{
			name:    "Source field not exists",
			json:    `{"j1": true, "j2": false}`,
			source:  "not_exist",
			wantErr: true,
		},
		{
			name: "Field name not exists",
			json: `{"j1": true, "j2": false}`,
			field: map[string]string{
				"j1": "not_exist",
				"j2": "f2",
			},
			wantErr: true,
		},
		{
			name:      "TrueFalseValues error",
			json:      `{"j1": true, "j2": false}`,
			trueFalse: []string{"true", "false", "other"},
			wantErr:   true,
		},
	}

	fieldByName := func(name string) (baker.FieldIndex, bool) {
		switch name {
		case "f1":
			return 0, true
		case "f2":
			return 1, true
		case "f3":
			return 2, true
		case "j":
			return 3, true
		}
		return 0, false
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.field) == 0 {
				tt.field = map[string]string{
					"j1": "f1",
					"j2": "f2",
					"j3": "f3",
				}
			}
			if len(tt.source) == 0 {
				tt.source = "j"
			}
			jsonConfig := ExpandJSONConfig{
				Fields: tt.field,
				Source: tt.source,
			}
			if len(tt.trueFalse) != 0 {
				jsonConfig.TrueFalseValues = tt.trueFalse
			}
			cfg := baker.FilterParams{
				ComponentParams: baker.ComponentParams{
					FieldByName:   fieldByName,
					DecodedConfig: &jsonConfig,
				},
			}

			filter, err := NewExpandJSON(cfg)
			if (err != nil) != (tt.wantErr) {
				t.Fatalf("got error = %v, want error = %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}

			l := &baker.LogLine{FieldSeparator: ';'}
			i, _ := fieldByName("j")
			l.Set(i, []byte(tt.json))

			filter.Process(l, func(line baker.Record) {
				for k, v := range tt.want {
					i, _ := fieldByName(k)
					if !bytes.Equal(line.Get(i), []byte(v)) {
						t.Errorf("got %q, want %q", line.Get(i), v)
					}
				}
				// check that the json field is untouched
				i, _ := fieldByName("j")
				if !bytes.Equal(line.Get(i), []byte(tt.json)) {
					t.Errorf("got %q, want %q", line.Get(i), tt.json)
				}
			})
		})
	}
}
