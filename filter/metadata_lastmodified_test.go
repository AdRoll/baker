package filter

import (
	"testing"
	"time"

	"github.com/AdRoll/baker"
	"github.com/AdRoll/baker/input/inpututils"
)

func TestMetadataLastModified(t *testing.T) {
	ptr := func(t time.Time) *time.Time {
		return &t
	}

	tests := []struct {
		name         string
		lastmodified *time.Time
		dst          string
		want         string
		wantErr      bool
	}{
		{
			name:         "time set",
			lastmodified: ptr(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)),
			dst:          "f2",
			want:         "1257894000",
		},
		{
			name:         "time set at zero",
			lastmodified: &time.Time{},
			dst:          "f2",
			want:         "",
		},
		{
			name:         "time not set",
			lastmodified: nil,
			dst:          "f2",
			want:         "",
		},

		// errors
		{
			name:    "DstField error",
			dst:     "not-exist",
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
			f, err := NewMetadataLastModified(baker.FilterParams{
				ComponentParams: baker.ComponentParams{
					FieldByName: fieldByName,
					DecodedConfig: &MetadataLastModifiedConfig{
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
			var bakerMetadata baker.Metadata
			if tt.lastmodified != nil {
				bakerMetadata = baker.Metadata{inpututils.MetadataLastModified: *tt.lastmodified}
			}
			if err := rec1.Parse([]byte{}, bakerMetadata); err != nil {
				t.Fatalf("parse error: %q", err)
			}

			f.Process(rec1, func(rec2 baker.Record) {
				id, ok := fieldByName(tt.dst)
				if !ok {
					t.Fatalf("cannot find field name")
				}
				unixTime := string(rec2.Get(id))
				if string(unixTime) != tt.want {
					t.Errorf("got UnixTime %q, want %q", unixTime, tt.want)
				}
			})
		})
	}
}
