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
		l           int
		start       int
		record      []byte
		want        []byte
		wantInitErr bool
		wantErr     bool
	}{
		{
			name:        "empty src",
			src:         "",
			dst:         "2nd",
			l:           5,
			wantInitErr: true,
		},
		{
			name:        "empty dst",
			src:         "1st",
			dst:         "",
			l:           5,
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
			l:           -2,
			wantInitErr: true,
		},
		{
			name:   "index over length",
			src:    "1st",
			dst:    "2nd",
			l:      5,
			start:  5,
			record: []byte("12345,b,c"),
			want:   []byte("12345,,c"),
		},
		{
			name:   "index over length, same field",
			src:    "1st",
			dst:    "1st",
			l:      5,
			start:  5,
			record: []byte("12345,b,c"),
			want:   []byte(",b,c"),
		},
		{
			name:   "Nothing to Slice, length <= field length",
			src:    "1st",
			dst:    "2nd",
			l:      5,
			record: []byte("12345,b,c"),
			want:   []byte("12345,12345,c"),
		},
		{
			name:   "Nothing to Slice, length > field length",
			src:    "1st",
			dst:    "2nd",
			l:      7,
			record: []byte("12345,b,c"),
			want:   []byte("12345,12345,c"),
		},
		{
			name:   "Sliced w/o start",
			src:    "1st",
			dst:    "2nd",
			l:      5,
			record: []byte("1234567890,b,c"),
			want:   []byte("1234567890,12345,c"),
		},
		{
			name:   "Sliced w start",
			src:    "1st",
			dst:    "2nd",
			start:  2,
			l:      5,
			record: []byte("1234567890,b,c"),
			want:   []byte("1234567890,34567,c"),
		},
		{
			name:   "Sliced w start, same field",
			src:    "1st",
			dst:    "1st",
			start:  3,
			l:      4,
			record: []byte("1234567890,b,c"),
			want:   []byte("4567,b,c"),
		},
		{
			name:   "Sliced w start, last field",
			src:    "1st",
			dst:    "3rd",
			start:  1,
			l:      4,
			record: []byte("1234567890,b,c"),
			want:   []byte("1234567890,b,2345"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := baker.FilterParams{
				ComponentParams: baker.ComponentParams{
					DecodedConfig: &SliceConfig{
						Src:      tt.src,
						Dst:      tt.dst,
						Length:   tt.l,
						StartIdx: tt.start,
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
