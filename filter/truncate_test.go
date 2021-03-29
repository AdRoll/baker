package filter

import (
	"bytes"
	"testing"

	"github.com/AdRoll/baker"
)

func TestTruncate(t *testing.T) {
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
			l:           0,
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
			name:   "Nothing to truncate",
			src:    "1st",
			dst:    "2nd",
			l:      5,
			record: []byte("a,b,c"),
			want:   []byte("a,b,c"),
		},
		{
			name:   "Truncated",
			src:    "1st",
			dst:    "2nd",
			l:      5,
			record: []byte("1234567890,b,c"),
			want:   []byte("1234567890,12345,c"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := baker.FilterParams{
				ComponentParams: baker.ComponentParams{
					DecodedConfig: &TruncateConfig{
						Src:    tt.src,
						Dst:    tt.dst,
						Length: tt.l,
					},
					FieldByName: fieldByName,
				},
			}

			filter, err := NewTruncate(cfg)
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
					t.Errorf("%s - got: %s ; want: %s", tt.name, g, w)
				}
			}
		})
	}
}
