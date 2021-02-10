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
			want: map[string]string{
				"f1": "value1",
				"f2": "value 2",
				"f3": "1,[2],3,4{",
			},
		},
		{
			name: "string test2",
			json: `{"junk1": 234, "junk2": {"a":"hello","b":[true,2]}, "j1": "text"}`,
			want: map[string]string{
				"f1": "text",
			},
		},
		{
			name: "test numbers",
			json: `{"j1": 12, "j2": 0.15, "j3": 15.12312312312}`,
			want: map[string]string{
				"f1": "12",
				"f2": "0.15",
				"f3": "15.12312312312",
			},
		},
		{
			name: "test boolean 1",
			json: `{"j1": true, "j2": false}`,
			want: map[string]string{
				"f1": "true",
				"f2": "false",
			},
		},
		{
			name: "test boolean 2",
			json: `{"j1": true, "j2": false}`,
			want: map[string]string{
				"f1": "t",
				"f2": "f",
			},
			trueFalse: []string{"t", "f"},
		},
		{
			name: "other json types",
			json: `{"j1": null, "j2": [1,2,3,4], "j3": {"j1":"a", "j2":false}}`,
			want: map[string]string{
				"f1": "",
				"f2": "[1,2,3,4]",
				"f3": `{"j1":"a","j2":false}`,
			},
		},
		{
			name: "empty json",
			json: ``,
			want: map[string]string{
				"f1": "",
				"f2": "",
			},
		},
		{
			name: "json parse error",
			json: `{this is not a json]`,
			want: map[string]string{
				"f1": "",
				"f2": "",
			},
		},
		{
			name: "more complex JMESPath expression",
			json: `[{"name": "name1"}, {"name": "name2"}]`,
			field: map[string]string{
				"[0].name": "f1",
				"[1].name": "f2",
			},
			want: map[string]string{
				"f1": "name1",
				"f2": "name2",
			},
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
			name: "JMESPath malformed",
			json: `{"j1": true, "j2": false}`,
			field: map[string]string{
				"j1":  "f1",
				"js.": "f2",
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
			if tt.source == "" {
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

			rec := &baker.LogLine{FieldSeparator: ';'}
			i, _ := fieldByName("j")
			rec.Set(i, []byte(tt.json))

			filter.Process(rec, func(rec2 baker.Record) {
				for k, v := range tt.want {
					i, _ := fieldByName(k)
					if !bytes.Equal(rec2.Get(i), []byte(v)) {
						t.Errorf("got %q, want %q", rec2.Get(i), v)
					}
				}
			})
		})
	}
}

func BenchmarkExpandJSON(b *testing.B) {
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
	cfg := baker.FilterParams{
		ComponentParams: baker.ComponentParams{
			FieldByName: fieldByName,
			DecodedConfig: &ExpandJSONConfig{
				Fields: map[string]string{
					"j1": "f1",
					"j2": "f2",
					"j3": "f3",
				},
				Source: "j",
			},
		},
	}
	filter, err := NewExpandJSON(cfg)
	if err != nil {
		b.Fatal(err)
	}
	rec := &baker.LogLine{FieldSeparator: ';'}
	i, _ := fieldByName("j")
	rec.Set(i, []byte(`{"j1": "value1", "j2": "value 2", "j3": "value3"}`))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filter.Process(rec, func(rec2 baker.Record) {
		})
	}
}
