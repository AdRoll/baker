package filter

import (
	"bytes"
	"testing"

	"github.com/AdRoll/baker"
)

func TestSlice(t *testing.T) {
	fields := map[string]baker.FieldIndex{
		"1st": 0,
		"2nd": 1,
		"3rd": 2,
	}

	fieldByName := func(s string) (baker.FieldIndex, bool) {
		idx, ok := fields[s]
		return idx, ok
	}

	tests := []struct {
		name        string
		src         string
		dst         string
		start       int
		end         int
		record      []byte
		want        []byte
		wantInitErr bool
		wantErr     bool
	}{
		{
			name:        "empty src",
			src:         "",
			dst:         "2nd",
			end:         5,
			wantInitErr: true,
		},
		{
			name:        "empty dst",
			src:         "1st",
			dst:         "",
			end:         5,
			wantInitErr: true,
		},
		{
			name:        "wrong length",
			src:         "1st",
			dst:         "2nd",
			wantInitErr: true,
		},
		{
			name:        "negative length",
			src:         "1st",
			dst:         "2nd",
			end:         -2,
			wantInitErr: true,
		},
		{
			name:   "Nothing to Slice, end <= field length",
			src:    "1st",
			dst:    "2nd",
			end:    5,
			record: []byte("12345,b,c"),
			want:   []byte("12345,12345,c"),
		},
		{
			name:   "Nothing to Slice, end > field length",
			src:    "1st",
			dst:    "2nd",
			end:    7,
			record: []byte("12345,b,c"),
			want:   []byte("12345,12345,c"),
		},
		{
			name:   "big end value",
			src:    "1st",
			dst:    "2nd",
			end:    100,
			record: []byte("12345,b,c"),
			want:   []byte("12345,12345,c"),
		},
		{
			name:   "Sliced w/o start",
			src:    "1st",
			dst:    "2nd",
			end:    5,
			record: []byte("1234567890,b,c"),
			want:   []byte("1234567890,12345,c"),
		},
		{
			name:   "Sliced w start",
			src:    "1st",
			dst:    "2nd",
			start:  2,
			end:    7,
			record: []byte("1234567890,b,c"),
			want:   []byte("1234567890,34567,c"),
		},
		{
			name:   "Sliced w start, same field",
			src:    "1st",
			dst:    "1st",
			start:  3,
			end:    7,
			record: []byte("1234567890,b,c"),
			want:   []byte("4567,b,c"),
		},
		{
			name:   "Sliced w start, last field",
			src:    "1st",
			dst:    "3rd",
			start:  1,
			end:    5,
			record: []byte("1234567890,b,c"),
			want:   []byte("1234567890,b,2345"),
		},
		{
			name:   "missing end",
			src:    "1st",
			dst:    "3rd",
			start:  1,
			record: []byte("1234567890,b,c"),
			want:   []byte("1234567890,b,234567890"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := baker.FilterParams{
				ComponentParams: baker.ComponentParams{
					DecodedConfig: &SliceConfig{
						Src:      tt.src,
						Dst:      tt.dst,
						StartIdx: tt.start,
						EndIdx:   tt.end,
					},
					FieldByName: fieldByName,
				},
			}

			filter, err := NewSlice(cfg)
			if err != nil && !tt.wantInitErr {
				t.Fatal(err)
			}

			if tt.wantInitErr {
				return
			}

			ll := &baker.LogLine{FieldSeparator: ','}
			if err := ll.Parse(tt.record, nil); err != nil {
				t.Fatal(err)
			}

			filter.Process(ll, func(baker.Record) {})

			want := &baker.LogLine{FieldSeparator: ','}
			if err := want.Parse(tt.want, nil); err != nil {
				t.Fatal(err)
			}

			// Compare records field by field
			for _, idx := range fields {
				g := ll.Get(idx)
				w := want.Get(idx)
				if !bytes.Equal(g, w) {
					t.Errorf("%s - got: %s ; want: %s (full %s)", tt.name, g, w, ll.ToText(nil))
				}
			}
		})
	}
}
