package filter

import (
	"bytes"
	"testing"

	"github.com/AdRoll/baker"
)

func TestHash(t *testing.T) {
	tests := []struct {
		name    string
		record  string
		src     string
		dst     string
		hash    string
		encode  string
		want    []byte
		wantErr bool
	}{
		{
			name:   "md5",
			record: "abc,def,ghi",
			src:    "f1",
			dst:    "f3",
			hash:   "md5",
			encode: "",
			want:   []byte{144, 1, 80, 152, 60, 210, 79, 176, 214, 150, 63, 125, 40, 225, 127, 114},
		},
		{
			name:   "md5 + hex",
			record: "abc,def,ghi",
			src:    "f1",
			dst:    "f3",
			hash:   "md5",
			encode: "hex",
			want:   []byte{57, 48, 48, 49, 53, 48, 57, 56, 51, 99, 100, 50, 52, 102, 98, 48, 100, 54, 57, 54, 51, 102, 55, 100, 50, 56, 101, 49, 55, 102, 55, 50},
		},
		{
			name:   "sha256",
			record: "abc,def,ghi",
			src:    "f1",
			dst:    "f3",
			hash:   "sha256",
			encode: "",
			want:   []byte{186, 120, 22, 191, 143, 1, 207, 234, 65, 65, 64, 222, 93, 174, 34, 35, 176, 3, 97, 163, 150, 23, 122, 156, 180, 16, 255, 97, 242, 0, 21, 173},
		},
		{
			name:   "sha256 + hex",
			record: "abc,def,ghi",
			src:    "f1",
			dst:    "f3",
			hash:   "sha256",
			encode: "hex",
			want:   []byte{98, 97, 55, 56, 49, 54, 98, 102, 56, 102, 48, 49, 99, 102, 101, 97, 52, 49, 52, 49, 52, 48, 100, 101, 53, 100, 97, 101, 50, 50, 50, 51, 98, 48, 48, 51, 54, 49, 97, 51, 57, 54, 49, 55, 55, 97, 57, 99, 98, 52, 49, 48, 102, 102, 54, 49, 102, 50, 48, 48, 49, 53, 97, 100},
		},

		// errors
		{
			name:    "Function error",
			src:     "f1",
			dst:     "f3",
			hash:    "hash-not-exist",
			encode:  "hex",
			wantErr: true,
		},
		{
			name:    "Encoding error",
			src:     "f1",
			dst:     "f3",
			hash:    "md5",
			encode:  "encoding-not-exist",
			wantErr: true,
		},
		{
			name:    "SrcField error",
			src:     "not-exist",
			dst:     "f1",
			hash:    "md5",
			encode:  "hex",
			wantErr: true,
		},
		{
			name:    "DstField error",
			src:     "f1",
			dst:     "not-exist",
			hash:    "md5",
			encode:  "hex",
			wantErr: true,
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
		}
		return 0, false
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := NewHash(baker.FilterParams{
				ComponentParams: baker.ComponentParams{
					FieldByName: fieldByName,
					DecodedConfig: &HashConfig{
						Function: tt.hash,
						Encoding: tt.encode,
						SrcField: tt.src,
						DstField: tt.dst,
					},
				},
			})

			if (err != nil) != (tt.wantErr) {
				t.Fatalf("got error = %v, want error = %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			rec1 := &baker.LogLine{FieldSeparator: ','}
			if err := rec1.Parse([]byte(tt.record), nil); err != nil {
				t.Fatalf("parse error: %q", err)
			}

			f.Process(rec1, func(rec2 baker.Record) {
				id, ok := fieldByName(tt.dst)
				if !ok {
					t.Fatalf("cannot find field name")
				}
				hash := rec2.Get(id)
				if !bytes.Equal(hash, tt.want) {
					t.Errorf("got hash %q, want %q", hash, tt.want)
				}
			})
		})
	}
}
