package filter

import (
	"testing"

	"github.com/AdRoll/baker"
)

func TestDedup(t *testing.T) {
	tests := []struct {
		name    string
		records []string
		fields  []string
		want    int // number of output records
		wantErr bool
	}{
		{
			name: "all different",
			records: []string{
				"abc1,def1,ghi1",
				"abc2,def2,ghi2",
				"abc3,def3,ghi3",
			},
			fields: []string{"f1", "f2", "f3"},
			want:   3,
		},
		{
			name: "all equal",
			records: []string{
				"abc,def,ghi",
				"abc,def,ghi",
				"abc,def,ghi",
			},
			fields: []string{"f1", "f2", "f3"},
			want:   1,
		},
		{
			name: "1 field equal",
			records: []string{
				"abc,def1,ghi1",
				"abc,def2,ghi2",
				"abc,def3,ghi3",
			},
			fields: []string{"f1"},
			want:   1,
		},
		{
			name: "1 field different",
			records: []string{
				"abc,def1,ghi",
				"abc,def2,ghi",
				"abc,def3,ghi",
			},
			fields: []string{"f2"},
			want:   3,
		},

		// errors
		{
			name: "not existing field",
			records: []string{
				"abc,def,ghi",
			},
			fields:  []string{"not_exist"},
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
			params := baker.FilterParams{
				ComponentParams: baker.ComponentParams{
					FieldByName: fieldByName,
					DecodedConfig: &DedupConfig{
						Fields: tt.fields,
					},
				},
			}
			f1, err := NewDedup(params)
			if (err != nil) != (tt.wantErr) {
				t.Fatalf("got error = %v, want error = %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			var count int
			next := func(baker.Record) { count++ }

			for _, rec := range tt.records {
				l := &baker.LogLine{FieldSeparator: ','}
				if err := l.Parse([]byte(rec), nil); err != nil {
					t.Fatalf("parse error: %q", err)
				}
				f1.Process(l, next)
			}

			if count != tt.want {
				t.Errorf("got %d record, want %d", count, tt.want)
			}
		})
	}
}
