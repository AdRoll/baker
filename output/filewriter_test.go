package output

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/AdRoll/baker"
	"github.com/AdRoll/baker/pkg/zip_agnostic"
	"github.com/AdRoll/baker/testutil"
)

func TestFileWriterConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *FileWriterConfig
		fields  []baker.FieldIndex
		wantErr bool
	}{
		{
			name: "all defaults",
			cfg:  &FileWriterConfig{},
		},
		{
			name: "{{.Field0}} and len(output.fields) == 1",
			cfg: &FileWriterConfig{
				PathString: "/path/{{.Field0}}/file.gz",
			},
			fields: []baker.FieldIndex{0},
		},
		{
			name: "{{.Field0}} and len(output.fields) > 1",
			cfg: &FileWriterConfig{
				PathString: "/path/{{.Field0}}/file.gz",
			},
			fields: []baker.FieldIndex{0, 1},
		},

		// error cases
		{
			name: "{{.Field0}} and len(output.fields) == 0",
			cfg: &FileWriterConfig{
				PathString: "/path/{{.Field0}}/file.gz",
			},
			fields:  []baker.FieldIndex{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := baker.OutputParams{
				ComponentParams: baker.ComponentParams{
					DecodedConfig: tt.cfg,
				},
				Fields: tt.fields,
			}
			_, err := NewFileWriter(cfg)
			if tt.wantErr && err == nil {
				t.Fatalf("wantErr: %v, got: %v", tt.wantErr, err)
			}

			if !tt.wantErr && err != nil {
				t.Fatalf("wantErr: %v, got: %v", tt.wantErr, err)
			}
		})
	}
}

// testFileWriterCompareInOut sends numRecords records and wait for the given
// duration between each send, to a FileWriter where PathString is set to the
// concatenation of the path components.
func testFileWriterCompareInOut(numRecords int, wait, rotate time.Duration, comps ...string) func(*testing.T) {
	return func(t *testing.T) {
		cfg := baker.OutputParams{
			Fields: []baker.FieldIndex{1},
			ComponentParams: baker.ComponentParams{
				DecodedConfig: &FileWriterConfig{
					PathString:     filepath.Join(append([]string{t.TempDir()}, comps...)...),
					RotateInterval: rotate,
				},
			},
		}
		fw, err := NewFileWriter(cfg)
		if err != nil {
			t.Fatal(err)
		}

		// Send records to inch and keep track of them in a map for later comparison.
		inch := make(chan baker.OutputRecord)
		sentRecords := make(map[int]string)
		go func() {
			for i := 0; i < numRecords; i++ {
				record := fmt.Sprintf("foo,%d,bar", i)
				inch <- baker.OutputRecord{Fields: []string{strconv.Itoa(i)}, Record: []byte(record)}
				sentRecords[i] = record
				time.Sleep(wait)
			}
			close(inch)
		}()

		upch := make(chan string)
		errc := make(chan error, 1)
		go func() {
			errc <- fw.Run(inch, upch)
			close(upch)
		}()

		// Drain the channel containing the uploaded paths.
		var uploaded []string
		for p := range upch {
			uploaded = append(uploaded, p)
		}

		if err := <-errc; err != nil {
			t.Fatalf("fw.Run() error: %v", err)
		}

		// Verify that the set of records sent to the output is equal to the set of
		// records present in the file(s) sent to the uploader.
		uploadedRecords := make(map[int]string)

		for _, p := range uploaded {
			f, err := os.Open(p)
			if err != nil {
				t.Fatalf("can't open uploaded path: %s", err)
			}
			defer f.Close()

			fi, err := os.Stat(p)
			if err != nil {
				t.Fatalf("can't stat uploaded path: %s", err)
			}

			t.Logf("%s size: %d", p, fi.Size())
			if fi.Size() == 0 {
				continue
			}

			zr, err := zip_agnostic.NewReader(f)
			if err != nil {
				t.Fatalf("can't open uploaded path: %s", err)
			}
			defer zr.Close()

			s := bufio.NewScanner(zr)
			rec := baker.LogLine{FieldSeparator: ','}
			for s.Scan() {
				if err := rec.Parse(s.Bytes(), nil); err != nil {
					t.Fatalf("can't scan record: %s: %v", s.Text(), err)
				}
				idx, err := strconv.Atoi(string(rec.Get(1)))
				if err != nil {
					t.Fatalf("field 0 in %q: %v", s.Bytes(), err)
				}
				uploadedRecords[idx] = s.Text()
			}
			if err := s.Err(); err != nil {
				t.Fatalf("%s: scan error: %v", p, err)
			}
		}

		if !reflect.DeepEqual(sentRecords, uploadedRecords) {
			t.Errorf("total mismatch: %d records found in uploaded files, sent %d", len(uploadedRecords), len(sentRecords))
		}
	}
}

func TestFileWriterCompareInOut(t *testing.T) {
	t.Parallel()

	defer testutil.DisableLogging()()

	tests := []struct {
		name       string
		numRecords int
		wait       time.Duration
		rotate     time.Duration
		comps      []string
	}{
		{
			name:       "year-month-rotation/out=gz",
			numRecords: 500,
			wait:       1 * time.Millisecond,
			rotate:     time.Second,
			comps:      []string{"{{.Year}}", "{{.Month}}", "{{.Rotation}}-out.csv.gz"},
		},
		{
			name:       "year-month/out=gz",
			numRecords: 500,
			wait:       0,
			rotate:     time.Second,
			comps:      []string{"{{.Year}}", "{{.Month}}", "out.csv.gz"},
		},
		{
			name:       "year-month-rotation/out=zst",
			numRecords: 500,
			wait:       1 * time.Millisecond,
			rotate:     time.Second,
			comps:      []string{"{{.Year}}", "{{.Month}}", "{{.Rotation}}-out.csv.zst"},
		},
		{
			name:       "year-month/out=zst",
			numRecords: 500,
			wait:       0,
			rotate:     time.Second,
			comps:      []string{"{{.Year}}", "{{.Month}}", "out.csv.zst"},
		},
		{
			name:       "disable-rotation",
			numRecords: 500,
			wait:       0,
			rotate:     -1,
			comps:      []string{"disable-rotation.out.csv.zst"},
		},
		{
			name:       "year-month/field0-out=zst",
			numRecords: 20,
			wait:       1 * time.Millisecond,
			rotate:     time.Second,
			comps:      []string{"{{.Year}}", "{{.Month}}", "{{.Field0}}-out.csv.zst"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, testFileWriterCompareInOut(tt.numRecords, tt.wait, tt.rotate, tt.comps...))
	}
}
