package filter

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/AdRoll/baker"
)

func TestExternalMatchConfigFillDefaults(t *testing.T) {
	tests := []struct {
		name    string
		cfg     ExternalMatchConfig
		wantErr bool
	}{
		{
			name: "ok",
			cfg: ExternalMatchConfig{
				Files:          []string{"file:///path/to/%s/file.csv"},
				DateTimeLayout: "2006",
			},
			wantErr: false,
		},
		{
			name: "ok",
			cfg: ExternalMatchConfig{
				Files:          []string{"file:///path/to/%s/file.csv"},
				DateTimeLayout: "2006",
				TimeSubtract:   24,
			},
			wantErr: false,
		},
		{
			name:    "ok",
			cfg:     ExternalMatchConfig{Files: []string{"file:///path/to/file.csv"}},
			wantErr: false,
		},

		// error cases
		{
			name: "unsupported scheme http",
			cfg: ExternalMatchConfig{
				Files: []string{"http://github.com"},
			},
			wantErr: true,
		},
		{
			name: "invalid url",
			cfg: ExternalMatchConfig{
				Files: []string{"s3://::github.com"},
			},
			wantErr: true,
		},
		{
			name: "%s with DateTimeLayout not set",
			cfg: ExternalMatchConfig{
				Files: []string{"file:///path/to/%s/file.csv"},
			},
			wantErr: true,
		},
		{
			name: "%s with DateTimeLayout not set",
			cfg: ExternalMatchConfig{
				Files: []string{
					"file:///path/to/file.csv",
					"file:///path/to/%s/file.csv",
				},
			},
			wantErr: true,
		},
		{
			name: "DateTimeLayout with no %s in Files",
			cfg: ExternalMatchConfig{
				Files:          []string{"file:///path/to/file.csv"},
				DateTimeLayout: "2006",
			},
			wantErr: true,
		},
		{
			name: "DateTimeLayout with no %s in Files",
			cfg: ExternalMatchConfig{
				Files: []string{
					"file:///path/to/%s/file.csv",
					"file:///path/to/file.csv",
				},
				DateTimeLayout: "2006",
			},
			wantErr: true,
		},
		{
			name: "TimeSubtract without DateTimeLayout",
			cfg: ExternalMatchConfig{
				Files:        []string{"file:///path/to/file.csv"},
				TimeSubtract: 1,
			},
			wantErr: true,
		},
		{
			name: "negative column",
			cfg: ExternalMatchConfig{
				Files:     []string{"file:///path/to/file.csv"},
				CSVColumn: -2,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.cfg.fillDefaults(); (err != nil) != tt.wantErr {
				t.Errorf("fillDefaults returns %v, want error: %t", err, tt.wantErr)
			}
		})
	}
}

func TestExternalMatchConfig_evaluateURLs(t *testing.T) {
	now := time.Now()

	tests := []struct {
		url            string
		dateTimeLayout string
		timeSubtract   time.Duration
		want           string
	}{
		{
			url:            "s3://%s",
			dateTimeLayout: "2006/01/02",
			want:           fmt.Sprintf("s3://%s", now.Format("2006/01/02")),
		},
		{
			url:            "s3://%s",
			dateTimeLayout: "2006/01/02",
			timeSubtract:   24 * time.Hour,
			want:           fmt.Sprintf("s3://%s", now.AddDate(0, 0, -1).Format("2006/01/02")),
		},
		{
			url:  "s3n://foo/bar/baz.csv",
			want: "s3n://foo/bar/baz.csv",
		},
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			cfg := &ExternalMatchConfig{
				Files:          []string{tt.url},
				DateTimeLayout: tt.dateTimeLayout,
				TimeSubtract:   tt.timeSubtract,
			}
			if got := cfg.evaluateURLs(); got[0] != tt.want {
				t.Errorf("ExternalMatchConfig.evaluateURLs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func testExternalMatchFiles(t *testing.T, keepOnMatch bool, fields []string, want []bool) {
	t.Helper()

	if len(fields) != len(want) {
		panic("len(fields) != len(want)")
	}

	// This test helper runs the ExternalMatch filter with a predefined
	// configuration that makes it loads the CSV containing 'content' and
	// DiscardOnMatch is set to discardOnMatch. Each record passed to the filter
	// has the value of FieldName set to values of 'fields', while 'want'
	// indicates whether we expect the filter to forward (true) or discard
	// (false) them.
	const content = `
row-0-col-0,row-0-col-1,AAAA,row-0-col-2,
row-1-col-0,row-1-col-1,"BBBB",row-1-col-2,
row-2-col-0,row-2-col-1,CCCC,row-2-col-2,
row-2-col-0,row-2-col-1
`

	const fieldIndex = 23

	// Our time reference for this test.
	now := time.Now()
	tmpDir := t.TempDir()

	// We use 'last month' rather than yesterday to avoid false negatives in
	// case the execution of this test starts at 23:59:xx. Indeed we could
	// potentially not be the same day when the test starts than when the filter
	// runs. This could still happen if the the test runs at 23:59:xx the last
	// day of the month but it's a risk we'll assume :-).
	padInt := func(i int) string { return fmt.Sprintf("%02d", i) }
	lastMonth := now.AddDate(0, -1, 0)
	tmpPath := filepath.Join(tmpDir, padInt(lastMonth.Year()), padInt(int(lastMonth.Month())), padInt(lastMonth.Day()), "values.csv")
	if err := os.MkdirAll(filepath.Dir(tmpPath), os.ModePerm); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tmpPath, []byte(content), os.ModePerm); err != nil {
		t.Fatal(err)
	}

	cfg := ExternalMatchConfig{
		Files:          []string{pathToURI(filepath.Join(tmpDir, "%s", "values.csv"))},
		DateTimeLayout: "2006/01/02",
		// Compute number of hours between now and lastmonth.
		TimeSubtract: now.Sub(lastMonth),
		CSVColumn:    2,
		KeepOnMatch:  keepOnMatch,
		FieldName:    "field",
		RefreshEvery: time.Hour,
	}

	iface, err := NewExternalMatch(baker.FilterParams{
		ComponentParams: baker.ComponentParams{
			DecodedConfig: &cfg,
			FieldByName: func(fname string) (baker.FieldIndex, bool) {
				if fname != "field" {
					panic("unexpected field name")
				}
				return fieldIndex, true
			},
		},
	})
	if err != nil {
		t.Fatalf("can't create filter: %v", err)
	}
	f := iface.(*ExternalMatch)

	bool2KeptOrDiscarded := func(b bool) string {
		if b {
			return "kept"
		}
		return "discarded"
	}

	for i, v := range fields {
		rec := baker.LogLine{}
		rec.Set(fieldIndex, []byte(v))
		got := false
		f.Process(&rec, func(r baker.Record) { got = true })
		if got != want[i] {
			t.Errorf("after Process(), the record with field %q has been %s, want %s", v, bool2KeptOrDiscarded(got), bool2KeptOrDiscarded(want[i]))
		}
	}
}

func TestExternalMatchDiscardOnMatch(t *testing.T) {
	t.Parallel()

	testExternalMatchFiles(t, false,
		[]string{"AAAA", "0000", "BBBB", "CCCC", "DDDD", "1111"},
		[]bool{false, true, false, false, true, true},
	)
}
func TestExternalMatchKeepOnMatch(t *testing.T) {
	t.Parallel()

	testExternalMatchFiles(t, true,
		[]string{"AAAA", "0000", "BBBB", "CCCC", "DDDD", "1111"},
		[]bool{true, false, true, true, false, false},
	)
}

func TestExternalMatchRefreshValues(t *testing.T) {
	t.Parallel()

	const refreshEvery = 1300 * time.Millisecond

	tmpDir := t.TempDir()
	tmpPath := filepath.Join(tmpDir, "values.csv")
	values := "AAAA"

	// Our reference record. First time we call Process it should be kept. Next
	// time we call Process, however, it should get discarded since the CSV file
	// won't contain 'AAAA' and, given the RefreshEvery parameter, the filter
	// internal state will have been updated to reflect that.
	rec := baker.LogLine{}
	rec.Set(0, []byte("AAAA"))

	// Create first version of the file.
	if err := os.WriteFile(tmpPath, []byte(values), os.ModePerm); err != nil {
		t.Fatal(err)
	}

	iface, err := NewExternalMatch(baker.FilterParams{
		ComponentParams: baker.ComponentParams{
			FieldByName: func(fname string) (baker.FieldIndex, bool) { return 0, true },
			DecodedConfig: &ExternalMatchConfig{
				Files:        []string{pathToURI(tmpPath)},
				FieldName:    "field",
				RefreshEvery: refreshEvery,
				KeepOnMatch:  true,
			},
		},
	})
	if err != nil {
		t.Fatalf("can't create filter: %v", err)
	}
	f := iface.(*ExternalMatch)
	t.Logf("filter created")

	kept := false
	f.Process(&rec, func(r baker.Record) { kept = true })
	if !kept {
		t.Error("first call to Process() has discarded the record, should have been kept")
	}

	// Create second version of the file.
	if err := os.WriteFile(tmpPath, []byte("BBBB"), os.ModePerm); err != nil {
		t.Fatal(err)
	}

	// Wait twice the amount of time after which the file should have been
	// refreshed, to account for distrorted CPU time on shared CI machines.
	time.Sleep(2 * refreshEvery)

	kept = false
	f.Process(&rec, func(r baker.Record) { kept = true })
	if kept {
		t.Error("second call to Process() has kept the record, should have been discarded")
	}
}

func pathToURI(p string) string {
	return "file://" + filepath.ToSlash(p)
}
