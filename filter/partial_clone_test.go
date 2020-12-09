package filter

import (
	"testing"

	"github.com/AdRoll/baker"
)

func TestNewPartialClone(t *testing.T) {
	fieldByName := func(fname string) (baker.FieldIndex, bool) {
		switch fname {
		case "field0":
			return 0, true
		case "field1":
			return 1, true
		case "field2":
			return 2, true
		default:
			return -1, false
		}
	}

	tests := []struct {
		name         string
		confFields   []string
		wantFieldIdx []baker.FieldIndex
		wantErr      bool
		record       string
		wantRecord   []string
	}{
		{
			name:         "base conf",
			confFields:   []string{"field0", "field2"},
			wantFieldIdx: []baker.FieldIndex{0, 2},
			record:       "a,foo,bar",
			wantRecord:   []string{"a", "", "bar"},
		},
		{
			name:         "empty conf",
			confFields:   []string{},
			wantFieldIdx: []baker.FieldIndex{},
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := baker.FilterParams{
				ComponentParams: baker.ComponentParams{
					CreateRecord: func() baker.Record { return &baker.LogLine{FieldSeparator: ','} },
					FieldByName:  fieldByName,
					DecodedConfig: &PartialCloneConfig{
						Fields: tt.confFields,
					},
				},
			}

			filter, err := NewPartialClone(params)
			if (err == nil) == tt.wantErr {
				t.Fatalf("err: %v, wantErr: %t", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			if len(filter.(*PartialClone).fieldIdx) != len(tt.confFields) {
				t.Errorf("fields len, want: %d, got: %d", len(tt.confFields), len(filter.(*PartialClone).fieldIdx))
			}

			for _, idx := range tt.wantFieldIdx {
				got := false
				for _, fieldIdx := range filter.(*PartialClone).fieldIdx {
					if fieldIdx == idx {
						got = true
						break
					}
				}
				if !got {
					t.Errorf("can't find field %d in fieldIdx", idx)
				}
			}

			var newRecord string
			nextFn := func(r baker.Record) {
				newRecord = string(r.(*baker.LogLine).ToText(nil))
			}

			record := &baker.LogLine{FieldSeparator: ','}
			record.Parse([]byte(tt.record), nil)

			filter.Process(record, nextFn)

			r := &baker.LogLine{FieldSeparator: ','}
			for i, v := range tt.wantRecord {
				r.Set(baker.FieldIndex(i), []byte(v))
			}
			if newRecord != string(r.ToText(nil)) {
				t.Errorf("record mismatch, want: %s got: %s", tt.wantRecord, newRecord)
			}
		})
	}
}
