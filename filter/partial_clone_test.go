package filter

import (
	"reflect"
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
		name               string
		discardNotMatching bool
		confMatches        [][]string
		confFields         []string
		wantFieldIdx       []baker.FieldIndex
		wantMatches        map[baker.FieldIndex][][]byte
	}{
		{
			name:        "base conf",
			confMatches: [][]string{[]string{"field0", "a"}, []string{"field1", "b"}, []string{"field2", "c"}},
			wantMatches: map[baker.FieldIndex][][]byte{
				0: [][]byte{[]byte("a")},
				1: [][]byte{[]byte("b")},
				2: [][]byte{[]byte("c")},
			},
			confFields:         []string{"field0", "field2"},
			wantFieldIdx:       []baker.FieldIndex{0, 2},
			discardNotMatching: true,
		},
		{
			name:        "complex conf",
			confMatches: [][]string{[]string{"field0", "a", "d", "efg"}, []string{"field1", "b", "1we%rty"}, []string{"field2", "c", "e"}},
			wantMatches: map[baker.FieldIndex][][]byte{
				0: [][]byte{[]byte("a"), []byte("d"), []byte("efg")},
				1: [][]byte{[]byte("b"), []byte("1we%rty")},
				2: [][]byte{[]byte("c"), []byte("e")},
			},
			confFields:         []string{"field1", "field0", "field2"},
			wantFieldIdx:       []baker.FieldIndex{1, 0, 2},
			discardNotMatching: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := baker.FilterParams{
				ComponentParams: baker.ComponentParams{
					FieldByName: fieldByName,
					DecodedConfig: &PartialCloneConfig{
						Matches:            tt.confMatches,
						Fields:             tt.confFields,
						DiscardNotMatching: tt.discardNotMatching,
					},
				},
			}

			filter, err := NewPartialClone(params)
			if err != nil {
				t.Fatalf("init error: %v", err)
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

			if filter.(*PartialClone).discardNotMatching != tt.discardNotMatching {
				t.Errorf("discardNotMatching, want: %t got: %t", tt.discardNotMatching, filter.(*PartialClone).discardNotMatching)
			}

			for i := range tt.confMatches {
				field := tt.confMatches[i][0]
				idx, _ := fieldByName(field)

				m, ok := filter.(*PartialClone).matches[idx]
				if !ok {
					t.Errorf("can't find key %s in matches", field)
				}

				if !reflect.DeepEqual(tt.wantMatches[idx], m) {
					t.Errorf("wrong map for field %s field, got: %v, want: %v", field, m, tt.wantMatches[idx])
				}
			}
		})
	}
}

func TestPartialClone_Process(t *testing.T) {
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
		name               string
		record             string
		discardNotMatching bool
		wantNext           bool
		wantRecord         string
	}{
		{
			name:               "doesn't match, calls next unchanged",
			record:             "notmatch,foo,bar",
			discardNotMatching: false,
			wantNext:           true,
			wantRecord:         "notmatch,foo,bar",
		},
		{
			name:               "doesn't match, doesn't calls next",
			record:             "notmatch,foo,bar",
			discardNotMatching: true,
			wantNext:           false,
		},
		{
			name:               "matches",
			record:             "a,foo,bar",
			discardNotMatching: true,
			wantNext:           true,
			wantRecord:         "a,,bar,,",
		},
	}

	params := baker.FilterParams{
		ComponentParams: baker.ComponentParams{
			CreateRecord: func() baker.Record { return &baker.LogLine{FieldSeparator: ','} },
			FieldByName:  fieldByName,
			DecodedConfig: &PartialCloneConfig{
				Matches: [][]string{[]string{"field0", "a", "b"}},
				Fields:  []string{"field0", "field2"},
			},
		},
	}
	filter, err := NewPartialClone(params)
	if err != nil {
		t.Fatalf("init error: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter.(*PartialClone).discardNotMatching = tt.discardNotMatching

			var called bool
			var newRecord string
			next := func(r baker.Record) {
				called = true
				newRecord = string(r.(*baker.LogLine).ToText(nil))
			}

			record := &baker.LogLine{FieldSeparator: ','}
			record.Parse([]byte(tt.record), nil)

			filter.Process(record, next)
			if called != tt.wantNext {
				t.Errorf("next, want: %t, got: %t", tt.wantNext, called)
			}

			if !tt.wantNext {
				return
			}

			if newRecord != tt.wantRecord {
				t.Errorf("record mismatch, want: %s got: %s", tt.wantRecord, newRecord)
			}
		})
	}
}

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
		t.Run(tt.name, func(t *testing.T) {

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
				t.Fatalf("init error: %v", err)
			}

			record := &baker.LogLine{FieldSeparator: ','}
			record.Parse([]byte(tt.record), nil)

			if filter.(*PartialClone).recordMatch(record) != tt.want {
				t.Errorf("\"%s\" match error", tt.name)
			}
		})

	}
}
