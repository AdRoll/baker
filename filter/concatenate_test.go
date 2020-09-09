package filter

import (
	"bytes"
	"testing"

	"github.com/AdRoll/baker"
)

func TestConcatenateFilter(t *testing.T) {
	tests := []struct {
		name      string
		fields    []string
		record    string
		separator string
		want      string
	}{
		{
			name:   "without separator",
			record: "v0,v1,v2,\n",
			fields: []string{"src2", "src0", "src1"},
			want:   "v2v0v1",
		},
		{
			name:      "with separator",
			record:    "v0,v1,v2,\n",
			fields:    []string{"src2", "src0", "src1"},
			separator: "~",
			want:      "v2~v0~v1",
		},
		{
			name:      "overwrites target",
			record:    "v0,v1,v2,trgt\n",
			fields:    []string{"src2", "src0", "src1"},
			separator: "~",
			want:      "v2~v0~v1",
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
		case "target":
			return 3, true
		}
		return 0, false
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			params := baker.FilterParams{
				ComponentParams: baker.ComponentParams{
					FieldByName: fieldByName,
					DecodedConfig: &ConcatenateConfig{
						Fields:    tt.fields,
						Target:    "target",
						Separator: tt.separator,
					},
				},
			}

			ll := baker.LogLine{FieldSeparator: ','}
			err := ll.Parse([]byte(tt.record), nil)
			if err != nil {
				t.Fatal(err)
			}

			f, err := NewConcatenate(params)
			if err != nil {
				t.Fatal(err)
			}

			f.Process(&ll, func(baker.Record) {})

			if !bytes.Equal(ll.Get(3), []byte(tt.want)) {
				t.Errorf("got: %q, want: %q", ll.Get(3), tt.want)
			}
		})
	}
}
