package filter

import (
	"testing"

	"github.com/AdRoll/baker"
)

func TestExpandList(t *testing.T) {
	tests := []struct {
		name string
		list string

		field  map[string]string
		source string // default: "source"
		sep    string // default: ";"

		want    map[string]string
		wantErr bool
	}{
		{
			name: "extract 2 value",
			list: "value1;value2",
			field: map[string]string{
				"0": "f1",
				"1": "f2",
			},
			want: map[string]string{
				"f1": "value1",
				"f2": "value2",
			},
		},
		{
			name: "extract 1 value",
			list: "value1;value2",
			field: map[string]string{
				"1": "f2",
			},
			want: map[string]string{
				"f1": "",
				"f2": "value2",
			},
		},
		{
			name: "sorce empty",
			list: "",
			field: map[string]string{
				"1": "f2",
			},
			want: map[string]string{
				"f1": "",
				"f2": "",
			},
		},
		{
			name: "out of range",
			list: "value1;value2",
			field: map[string]string{
				"93": "f2",
			},
			want: map[string]string{
				"f1": "",
				"f2": "",
			},
		},
		{
			name: "change separator",
			list: "value2-value1",
			field: map[string]string{
				"0": "f2",
				"1": "f1",
			},
			sep: "-",
			want: map[string]string{
				"f1": "value1",
				"f2": "value2",
			},
		},

		// errors
		{
			name:    "Source not exists",
			source:  "not_exist",
			wantErr: true,
		},
		{
			name: "Field name not exists",
			field: map[string]string{
				"0": "not_exist",
			},
			wantErr: true,
		},
		{
			name: "negative index",
			field: map[string]string{
				"-10": "f1",
			},
			wantErr: true,
		},
		{
			name: "index not a number",
			field: map[string]string{
				"foo": "f1",
			},
			wantErr: true,
		},
		{
			name:    "separator more 1-byte",
			field:   map[string]string{},
			sep:     "ab",
			wantErr: true,
		},
		{
			name:    "separator over max ASCII",
			field:   map[string]string{},
			sep:     string([]byte{132}),
			wantErr: true,
		},
	}

	fieldByName := func(name string) (baker.FieldIndex, bool) {
		switch name {
		case "f1":
			return 0, true
		case "f2":
			return 1, true
		case "source":
			return 2, true
		}
		return 0, false
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.source == "" {
				tt.source = "source"
			}
			cfg := baker.FilterParams{
				ComponentParams: baker.ComponentParams{
					FieldByName: fieldByName,
					DecodedConfig: &ExpandListConfig{
						Source:    tt.source,
						Fields:    tt.field,
						Separator: tt.sep,
					},
				},
			}

			filter, err := NewExpandList(cfg)
			if (err != nil) != (tt.wantErr) {
				t.Fatalf("got error = %v, want error = %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}

			rec := &baker.LogLine{FieldSeparator: ','}
			i, ok := fieldByName(tt.source)
			if !ok {
				t.Fatalf("uknown field %q", tt.source)
			}
			rec.Set(i, []byte(tt.list))

			filter.Process(rec, func(rec2 baker.Record) {
				for k, v := range tt.want {
					i, ok := fieldByName(k)
					if !ok {
						t.Fatalf("uknown field %q", k)
					}
					if string(rec2.Get(i)) != v {
						t.Errorf("got %q, want %q", rec2.Get(i), v)
					}
				}
			})
		})
	}
}
