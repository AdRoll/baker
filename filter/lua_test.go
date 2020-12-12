package filter

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/AdRoll/baker"
)

func BenchmarkLUAProcess(b *testing.B) {
	b.ReportAllocs()
	const script = `
-- rec is a record object
-- next is function next(record)
function dummy(rec, next)
    rec:set(0, "hey")
    next(rec)
end
`

	dir, err := ioutil.TempDir("", b.Name())
	if err != nil {
		b.Fatal(err)
	}
	// fname := filepath.Join(b.TempDir(), "filters.lua")
	fname := filepath.Join(dir, "filters.lua")
	if err := ioutil.WriteFile(fname, []byte(script), os.ModePerm); err != nil {
		b.Fatalf("can't write lua script: %v", err)
	}

	record := &baker.LogLine{}

	fieldByName := func(name string) (baker.FieldIndex, bool) {
		switch name {
		case "foo":
			return 0, true
		case "bar":
			return 1, true
		case "baz":
			return 2, true
		}
		return 0, false
	}

	f, err := NewLUA(baker.FilterParams{
		ComponentParams: baker.ComponentParams{
			FieldByName: fieldByName,
			DecodedConfig: &LUAConfig{
				Script:     fname,
				FilterName: "dummy",
			},
		},
	})

	if err != nil {
		b.Fatalf("NewLUA error = %v", err)
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		f.Process(record, func(baker.Record) {})
	}
}

func TestLUAFilter(t *testing.T) {
	// This is the lua script containing the lua functions used in the test cases.
	fname := filepath.Join("testdata", "lua_test.lua")

	fieldNames := []string{"foo", "bar", "baz"}
	fieldByName := func(name string) (baker.FieldIndex, bool) {
		for i, n := range fieldNames {
			if n == name {
				return baker.FieldIndex(i), true
			}
		}

		return 0, false
	}

	tests := []struct {
		name    string // both test case name and lua filter name
		record  string
		wantErr bool        // configuration-time error
		want    [][3]string // contains non discarded records with, for each of them, the 3 fields we want
	}{
		{
			name:   "swapFieldsWithIndex",
			record: "abc,def,ghi",
			want: [][3]string{
				{"abc", "ghi", "def"},
			},
		},
		{
			name:   "swapFieldsWithNames",
			record: "abc,def,ghi",
			want: [][3]string{
				{"abc", "ghi", "def"},
			},
		},
		{
			name:   "_createRecord",
			record: "abc,def,ghi",
			want: [][3]string{
				{"hey", "ho", "let's go!"},
				{"abc", "def", "ghi"},
			},
		},
		{
			name:   "_validateRecord",
			record: "ciao,,",
			want: [][3]string{
				{"good", "", ""},
			}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := NewLUA(baker.FilterParams{
				ComponentParams: baker.ComponentParams{
					FieldByName: fieldByName,
					FieldNames:  fieldNames,
					CreateRecord: func() baker.Record {
						return &baker.LogLine{FieldSeparator: ','}
					},
					ValidateRecord: func(r baker.Record) (bool, baker.FieldIndex) {
						if string(r.Get(0)) != "hello" {
							return false, 0
						}
						return true, -1
					},
					DecodedConfig: &LUAConfig{
						Script:     fname,
						FilterName: tt.name,
					},
				},
			})

			if (err != nil) != (tt.wantErr) {
				t.Fatalf("got error = %v, want error = %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			l := &baker.LogLine{FieldSeparator: ','}
			if err := l.Parse([]byte(tt.record), nil); err != nil {
				t.Fatalf("parse error: %q", err)
			}

			var got []baker.Record
			f.Process(l, func(r baker.Record) { got = append(got, r) })

			// Check the number of non discarded records match
			if len(got) != len(tt.want) {
				t.Fatalf("got %d non-discarded records, want %d", len(got), len(tt.want))
			}

			for recidx, rec := range tt.want {
				for fidx, fval := range rec {
					f := got[recidx].Get(baker.FieldIndex(fidx))
					if !bytes.Equal(f, []byte(fval)) {
						t.Errorf("got record[%d].Get(%d) = %q, want %q", recidx, fidx, string(f), fval)
					}
				}
			}
		})
	}
}
