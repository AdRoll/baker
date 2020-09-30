// +build cgo_sqlite

package output

import (
	"database/sql"
	"fmt"
	"sync"
	"testing"

	"github.com/AdRoll/baker"
	"github.com/AdRoll/baker/testutil"
	_ "github.com/mattn/go-sqlite3"
)

func makeWriter(t *testing.T, raw bool, path string, truncate bool) baker.Output {
	t.Helper()

	var cfg interface{}

	if raw {
		cfg = &SQLiteRawWriterConfig{
			PathString:     path,
			TableName:      "lines",
			Clear:          truncate,
			RecordBlobName: "raw_record",
		}
	} else {
		cfg = &SQLiteWriterConfig{
			PathString: path,
			TableName:  "lines",
			Clear:      truncate,
		}
	}

	fieldByName := func(name string) (baker.FieldIndex, bool) {
		switch name {
		case "field0":
			return 0, true
		case "field1":
			return 1, true
		case "field2":
			return 2, true
		}
		return 0, false
	}

	fieldName := func(i baker.FieldIndex) string {
		switch i {
		case 0:
			return "field0"
		case 1:
			return "field1"
		case 2:
			return "field2"
		}
		return ""
	}

	params := baker.OutputParams{
		Fields: []baker.FieldIndex{0, 1, 2},
		Index:  0,
		ComponentParams: baker.ComponentParams{
			DecodedConfig: cfg,
			FieldByName:   fieldByName,
			FieldName:     fieldName,
		},
	}
	writer, err := newSQLiteWriter(raw)(params)
	if err != nil {
		t.Fatalf("SQLite writer creation failed: %s", err)
	}
	return writer
}

// Just check that creating an sqlite writer does not crash.
func TestSQLiteNewSQLiteWriter(t *testing.T)    { makeWriter(t, false, ":memory:", false) }
func TestSQLiteNewSQLiteRawWriter(t *testing.T) { makeWriter(t, true, ":memory:", false) }

func doOneRound(t *testing.T, path string, raw bool) {
	doOneRoundTruncate(t, path, false, raw)
}

func doOneRoundTruncate(t *testing.T, path string, truncate, raw bool) {
	t.Helper()

	writer := makeWriter(t, raw, path, truncate)

	wg := &sync.WaitGroup{}
	outch := make(chan baker.OutputRecord)
	upch := make(chan string)

	path_seen := false

	wg.Add(1)
	go func() {
		outch <- baker.OutputRecord{Fields: []string{"val12345", "1500000", "BLAH"}, Record: []byte("rawrecord0")}
		outch <- baker.OutputRecord{Fields: []string{"val12345", "1500001", "BLAH"}, Record: []byte("rawrecord1")}
		outch <- baker.OutputRecord{Fields: []string{"val12348", "1500005", "BLOH"}, Record: []byte("rawrecord2")}
		outch <- baker.OutputRecord{Fields: []string{"val12349", "1500007", "BLUH"}, Record: []byte("rawrecord3")}
		close(outch)
		for upchpath := range upch {
			if upchpath == path {
				path_seen = true
			}
		}
		wg.Done()
	}()
	writer.Run(outch, upch)
	close(upch)
	wg.Wait()

	if !path_seen {
		t.Fatalf("SQLite file was not sent to upch.")
	}
}

func assertRows(t *testing.T, fname string, want [][]string) {
	// this helper asserts we're going to find these rows and records in
	// the sqlite database at the given path, in a table called 'lines'.
	t.Helper()

	conn, err := sql.Open("sqlite3", fname)
	if err != nil {
		t.Fatalf("sqlite database %q not found: %s", fname, err)
	}
	defer conn.Close()

	rows, err := conn.Query("SELECT * FROM lines")
	if err != nil {
		t.Fatalf("failed select: %s", err)
	}
	defer rows.Close()

	cols := make([]interface{}, len(want[0]))
	for i := range cols {
		cols[i] = new(string)
	}

	irow := 0
	for rows.Next() {
		if irow+1 >= len(want) {
			// There are more rows than we expected, anyway consume and count
			// them all to report the actual number in the error message.
			irow++
			continue
		}

		if err := rows.Scan(cols...); err != nil {
			t.Fatalf("failed scan: %s", err)
		}
		for i := range cols {
			col := *(cols[i].(*string))
			if want[irow][i] != col {
				t.Errorf("row[%d][%d] = %q, want %q", irow, i, col, want[irow][i])
			}
		}
		irow++
	}

	if irow != len(want) {
		t.Errorf("got %d total rows, want %d", irow, len(want))
	}
}

func TestSQLiteInsertRows(t *testing.T) { testSQLiteInsertRows(t, false) }

func TestSQLiteRawInsertRows(t *testing.T) { testSQLiteInsertRows(t, true) }

func testSQLiteInsertRows(t *testing.T, raw bool) {
	// This test tests that the lines fed into the output actually
	// show up in the output file. raw indicates whether the SQLWriter
	// is a raw one, in which case we also want to test that raw log lines
	// have been inserted, in addition to the set of fields.
	fn, rm := testutil.TempFile(t)
	defer rm()

	doOneRound(t, fn, raw)

	var want [][]string
	want = append(want, []string{"val12345", "1500000", "BLAH"})
	want = append(want, []string{"val12345", "1500001", "BLAH"})
	want = append(want, []string{"val12348", "1500005", "BLOH"})
	want = append(want, []string{"val12349", "1500007", "BLUH"})

	if raw {
		for i := range want {
			want[i] = append(want[i], fmt.Sprintf("rawrecord%d", i))
		}
	}

	assertRows(t, fn, want)
}

func TestSQLiteNoTruncate(t *testing.T) {
	// Tests that running the output twice doesn't truncate the table
	// (E.g. if I want to run Baker and insert to existing files)
	fn, rm := testutil.TempFile(t)
	defer rm()

	doOneRound(t, fn, false)
	doOneRound(t, fn, false)

	conn, err := sql.Open("sqlite3", fn)
	if err != nil {
		t.Fatalf("Cannot open output sqlite3 after running writer. %s", err)
	}
	defer conn.Close()

	rows, err := conn.Query("SELECT field0, field1, field2 FROM lines")
	if err != nil {
		t.Fatalf("Cannot select data from sqlite3 after running writer. %s", err)
	}
	defer rows.Close()

	row_id := 0
	for rows.Next() {
		row_id++
	}

	if row_id != 8 {
		t.Fatalf("Expected 8 lines to be in sqlite3 file.")
	}
}

func TestSQLiteTruncate(t *testing.T) {
	// Test that, if I turn on truncating, the thing actually truncates.
	fn, rm := testutil.TempFile(t)
	defer rm()

	doOneRound(t, fn, false)
	doOneRoundTruncate(t, fn, true, false)

	conn, err := sql.Open("sqlite3", fn)
	if err != nil {
		t.Fatalf("Cannot open output sqlite3 after running writer. %s", err)
	}
	defer conn.Close()

	rows, err := conn.Query("SELECT field0, field1, field2 FROM lines")
	if err != nil {
		t.Fatalf("Cannot select data from sqlite3 after running writer. %s", err)
	}
	defer rows.Close()

	row_id := 0
	for rows.Next() {
		row_id++
	}

	if row_id != 4 {
		t.Fatalf("Expected 4 lines to be in sqlite3 file.")
	}
}

func TestSQLitePrePostCommands(t *testing.T) {
	// This test adds some pre-run and post-run SQLite commands and tests that
	// their effects are present after running the writer.
	fn, rm := testutil.TempFile(t)
	defer rm()

	config := SQLiteWriterConfig{
		PathString: fn,
		TableName:  "lines",
		PreRun:     []string{"CREATE TABLE footable ( v INT )", "INSERT INTO footable ( v ) VALUES ( 55 )"},
		PostRun:    []string{"INSERT INTO footable ( v ) VALUES ( 56 )", "CREATE INDEX footable_v ON footable ( v )"},
	}

	fieldByName := func(name string) (baker.FieldIndex, bool) {
		switch name {
		case "field0":
			return 0, true
		case "field1":
			return 1, true
		case "field2":
			return 2, true
		}
		return 0, false
	}

	fieldName := func(i baker.FieldIndex) string {
		switch i {
		case 0:
			return "field0"
		case 1:
			return "field1"
		case 2:
			return "field2"
		}
		return ""
	}

	cfg := baker.OutputParams{
		Fields: []baker.FieldIndex{0, 1, 2},
		Index:  0,
		ComponentParams: baker.ComponentParams{
			DecodedConfig: &config,
			FieldByName:   fieldByName,
			FieldName:     fieldName,
		},
	}
	writer, err := newSQLiteWriter(false)(cfg)
	if err != nil {
		t.Fatalf("SQLite writer creation failed: %s", err)
	}

	outch := make(chan baker.OutputRecord)
	upch := make(chan string)

	// We are not actually putting any lines in this test so close record
	// channel right away.
	close(outch)

	go func() {
		for range upch {
		}
	}()

	writer.Run(outch, upch)
	close(upch)

	conn, err := sql.Open("sqlite3", fn)
	if err != nil {
		t.Fatalf("Cannot open output sqlite3 after running writer. %s", err)
	}
	defer conn.Close()

	rows, err := conn.Query("SELECT v FROM footable")
	if err != nil {
		t.Fatalf("Cannot select data from sqlite3 after running writer. %s", err)
	}
	defer rows.Close()

	row_id := 0
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			t.Fatalf("Cannot scan rows from sqlite3 after running writer. %s", err)
		}
		if row_id == 0 && v != 55 {
			t.Fatalf("Unexpected value inserted in pre-post test.")
		}
		if row_id == 1 && v != 56 {
			t.Fatalf("Unexpected value inserted in pre-post test.")
		}
		row_id++
	}
	if row_id != 2 {
		t.Fatalf("Unexpected number of rows.")
	}
}

func TestSQLiteBadConfig(t *testing.T) {
	tests := []struct {
		name       string
		pathstring string
	}{
		{
			name:       "bad template",
			pathstring: "{{\x00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &SQLiteRawWriterConfig{
				PathString:     tt.pathstring,
				TableName:      "lines",
				Clear:          false,
				RecordBlobName: "blob",
			}

			params := baker.OutputParams{
				Fields: []baker.FieldIndex{1},
				Index:  0,
				ComponentParams: baker.ComponentParams{
					DecodedConfig: cfg,
					FieldByName:   func(string) (baker.FieldIndex, bool) { return 0, true },
					FieldName:     func(baker.FieldIndex) string { return "name" },
				},
			}
			_, err := newSQLiteWriter(true)(params)
			if err == nil {
				t.Fatalf("want error, got nil")
			}
		})
	}
}
