package filter

import (
	"bytes"
	"testing"

	"github.com/AdRoll/baker"
)

func TestCopyFields(t *testing.T) {
	tests := []struct {
		name      string
		fieldsMap []string
		record    string
		want      string
		wantErr   bool
	}{
		{
			name:   "default replacements",
			record: "s0,s1,s2,t0,t1,t2",
			want:   "s0,s1,s2,s0,t1,s2",
			fieldsMap: []string{
				"src0", "trgt0",
				"src2", "trgt2",
			},
			wantErr: false,
		},
		{
			name:   "same replacement on multiple targets",
			record: "s0,s1,s2,t0,t1,t2",
			want:   "s0,s1,s2,s0,t1,s0",
			fieldsMap: []string{
				"src0", "trgt0",
				"src0", "trgt2",
			},
			wantErr: false,
		},
		{
			name:   "unknown src",
			record: "",
			want:   "",
			fieldsMap: []string{
				"unknown", "trgt0",
				"src2", "trgt2",
			},
			wantErr: true,
		},
		{
			name:   "unknown target",
			record: "",
			want:   "",
			fieldsMap: []string{
				"src0", "trgt0",
				"src2", "unknown",
			},
			wantErr: true,
		},
		{
			name:   "same src and target",
			record: "",
			want:   "",
			fieldsMap: []string{
				"src0", "trgt0",
				"src2", "src2",
			},
			wantErr: true,
		},
		{
			name:   "wrong fields number",
			record: "",
			want:   "",
			fieldsMap: []string{
				"src0", "trgt0", "src2",
			},
			wantErr: true,
		},
	}

	fieldByName := func(name string) (baker.FieldIndex, bool) {
		switch name {
		case "src0":
			return 0, true
		case "src1":
			return 1, true
		case "src2":
			return 2, true
		case "trgt0":
			return 3, true
		case "trgt1":
			return 4, true
		case "trgt2":
			return 5, true
		}
		return 0, false
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := baker.FilterParams{
				ComponentParams: baker.ComponentParams{
					FieldByName: fieldByName,
					DecodedConfig: &CopyFieldsConfig{
						FieldsMap: tt.fieldsMap,
					},
				},
			}

			ll := baker.LogLine{FieldSeparator: ','}
			if err := ll.Parse([]byte(tt.record), nil); err != nil {
				t.Fatal(err)
			}

			f, err := NewCopyFields(params)
			if tt.wantErr {
				if err == nil {
					t.Fatal("Expected conf err")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}

			f.Process(&ll, func(baker.Record) {})

			got := ll.ToText(nil)

			wantLine := baker.LogLine{FieldSeparator: ','}
			if err := wantLine.Parse([]byte(tt.want), nil); err != nil {
				t.Fatalf("Cannot parse wanted record: %v", err)
			}
			// Trick to avoid ToText to use the fast path and thus avoid
			// adding missing separators for all the logline fields
			wantLine.Set(0, wantLine.Get(0))

			want := wantLine.ToText(nil)

			if !bytes.Equal(got, want) {
				t.Errorf("got: %s, want: %s", got, want)
			}
		})
	}
}
