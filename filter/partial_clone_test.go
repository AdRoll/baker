package filter

import (
	"testing"

	"github.com/AdRoll/baker"
)

// func TestPartialClone(t *testing.T) {
// 	tests := []struct {
// 		record  string
// 		fields  []string
// 		regexs  []string
// 		want    bool // true: kept, false: discarded
// 		wantErr bool
// 	}{
// 		{
// 			fields:  []string{"foo"},
// 			regexs:  nil,
// 			wantErr: true,
// 		},
// 	}
// }

func TestPartialClone_recordMatch(t *testing.T) {
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
		name    string
		record  string
		matches [][]string
		want    bool
	}{
		{
			name:    "all AND, matches",
			matches: [][]string{[]string{"field0", "a"}, []string{"field1", "b"}, []string{"field2", "c"}},
			record:  "a,b,c",
			want:    true,
		},
		{
			name:    "all AND, does not match",
			matches: [][]string{[]string{"field0", "a"}, []string{"field1", "b"}, []string{"field2", "c"}},
			record:  "a,b,d",
			want:    false,
		},
		{
			name:    "mixed, matches",
			matches: [][]string{[]string{"field0", "a", "d"}, []string{"field1", "b"}, []string{"field2", "c"}},
			record:  "d,b,c",
			want:    true,
		},
		{
			name:    "mixed, does not match",
			matches: [][]string{[]string{"field0", "a", "d"}, []string{"field1", "b", "e"}, []string{"field2", "c"}},
			record:  "d,f,c",
			want:    false,
		},
		{
			name:    "empty, matches",
			matches: [][]string{},
			record:  "a,b,c",
			want:    true,
		},
	}

	for _, tt := range tests {
		params := baker.FilterParams{
			ComponentParams: baker.ComponentParams{
				FieldByName: fieldByName,
				DecodedConfig: &PartialCloneConfig{
					Matches: tt.matches,
				},
			},
		}

		filter, err := NewPartialClone(params)
		if err != nil {
			t.Errorf("%s init error: %v", tt.name, err)
		}

		record := &baker.LogLine{FieldSeparator: ','}
		record.Parse([]byte(tt.record), nil)

		if filter.(*PartialClone).recordMatch(record) != tt.want {
			t.Errorf("\"%s\" match error", tt.name)
		}
	}
}
