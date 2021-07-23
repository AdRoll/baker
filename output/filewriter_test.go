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
				inch <- baker.OutputRecord{Fields: nil, Record: []byte(record)}
				sentRecords[i] = record
				time.Sleep(wait)
			}
			close(inch)
		}()

		// TODO(arl): when the race condition has been fixed,
		// TestFileWriterCompareInOut should pass with addSync = true. At this
		// point, we can remove addSync.
		//
		// Since upch is not buffered, the output should guarantee that once
		// fw.Run() returns, all filenames should have been sent into the upload
		// channel. Thus should be no need for additional additional
		// synchronization here.
		const addSync = false

		var uploaded []string
		upch := make(chan string)
		done := make(chan struct{})
		go func() {
			for p := range upch {
				t.Logf("sent to uploader: %s", p)
				uploaded = append(uploaded, p)
			}
			if addSync {
				close(done)
			}
		}()

		if err := fw.Run(inch, upch); err != nil {
			t.Fatal(err)
		}
		if addSync {
			close(upch)
			<-done
		}

		// Verify that the set of records sent to the output is equal to the set of
		// records present in the -possibly multiple- files sent to the uploader.
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
			t.Fatalf("mismatch between sent and uploaded records\nsent:     %+v\nuploaded: %+v\n", sentRecords, uploadedRecords)
		}
	}
}

func TestFileWriterCompareInOut(t *testing.T) {
	t.Parallel()

	defer testutil.DisableLogging()()

	t.Run("year-month-rotation/out=gz",
		testFileWriterCompareInOut(500, 1*time.Millisecond, time.Second,
			"{{.Year}}", "{{.Month}}", "{{.Rotation}}-out.csv.gz",
		))

	t.Run("year-month/out=gz",
		testFileWriterCompareInOut(500, 0, time.Second,
			"{{.Year}}", "{{.Month}}", "out.csv.gz",
		))

	t.Run("year-month-rotation/out=zst",
		testFileWriterCompareInOut(500, 1*time.Millisecond, time.Second,
			"{{.Year}}", "{{.Month}}", "{{.Rotation}}-out.csv.zst",
		))

	t.Run("year-month/out=zst",
		testFileWriterCompareInOut(500, 0, time.Second,
			"{{.Year}}", "{{.Month}}", "out.csv.zst",
		))
}
