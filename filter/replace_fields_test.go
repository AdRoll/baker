package filter

import (
	"bytes"
	"testing"

	"github.com/AdRoll/baker"
)

func TestReplaceFields(t *testing.T) {
	tests := []struct {
		name          string
		copyFields    []string
		replaceFields []string
		record        string
		want          string
		wantErr       bool
	}{
		{
			name:   "default copy",
			record: "s0,s1,s2,t0,t1,t2",
			want:   "s0,s1,s2,s0,t1,s2",
			copyFields: []string{
				"src0", "dest0",
				"src2", "dest2",
			},
			wantErr: false,
		},
		{
			name:   "default replacement",
			record: "s0,s1,s2,t0,t1,t2",
			want:   "s0,s1,s2,fix0,t1,fix2",
			replaceFields: []string{
				"fix0", "dest0",
				"fix2", "dest2",
			},
			wantErr: false,
		},
		{
			name:   "both",
			record: "s0,s1,s2,t0,t1,t2",
			want:   "s0,s1,s2,s0,fix1,s2",
			copyFields: []string{
				"src0", "dest0",
				"src2", "dest2",
			},
			replaceFields: []string{
				"fix1", "dest1",
			},
			wantErr: false,
		},
		{
			name:   "same copy on multiple destinations",
			record: "s0,s1,s2,t0,t1,t2",
			want:   "s0,s1,s2,s0,t1,s0",
			copyFields: []string{
				"src0", "dest0",
				"src0", "dest2",
			},
			wantErr: false,
		},
		{
			name:   "copy on same destinations",
			record: "s0,s1,s2,t0,t1,t2",
			want:   "",
			copyFields: []string{
				"src0", "dest0",
				"src1", "dest0",
			},
			wantErr: true,
		},
		{
			name:   "replacements with dest duplications",
			record: "s0,s1,s2,t0,t1,t2",
			want:   "",
			replaceFields: []string{
				"fixed0", "dest0",
				"fixed1", "dest0",
			},
			wantErr: true,
		},
		{
			name:   "same field as copy and replacement destination",
			record: "s0,s1,s2,t0,t1,t2",
			want:   "",
			copyFields: []string{
				"src0", "dest0",
			},
			replaceFields: []string{
				"fixed0", "dest0",
			},
			wantErr: true,
		},
		{
			name:   "unknown src",
			record: "",
			want:   "",
			copyFields: []string{
				"unknown", "dest0",
				"src2", "dest2",
			},
			wantErr: true,
		},
		{
			name:   "unknown destination",
			record: "",
			want:   "",
			copyFields: []string{
				"src0", "dest0",
				"src2", "unknown",
			},
			wantErr: true,
		},
		{
			name:   "same src and destination",
			record: "",
			want:   "",
			copyFields: []string{
				"src0", "dest0",
				"src2", "src2",
			},
			wantErr: true,
		},
		{
			name:   "wrong fields number",
			record: "",
			want:   "",
			copyFields: []string{
				"src0", "dest0", "src2",
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
		case "dest0":
			return 3, true
		case "dest1":
			return 4, true
		case "dest2":
			return 5, true
		}
		return 0, false
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := baker.FilterParams{
				ComponentParams: baker.ComponentParams{
					FieldByName: fieldByName,
					DecodedConfig: &ReplaceFieldsConfig{
						CopyFields:    tt.copyFields,
						ReplaceFields: tt.replaceFields,
					},
				},
			}

			ll := baker.LogLine{FieldSeparator: ','}
			if err := ll.Parse([]byte(tt.record), nil); err != nil {
				t.Fatal(err)
			}

			f, err := NewReplaceFields(params)
			if tt.wantErr {
				if err == nil {
					t.Fatal("Expected conf err")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}

			_ = f.Process(&ll)

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
