package output_test

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/arl/dirtree"

	"github.com/AdRoll/baker"
	"github.com/AdRoll/baker/input"
	"github.com/AdRoll/baker/output"
	"github.com/AdRoll/baker/pkg/zip_agnostic"
	"github.com/AdRoll/baker/testutil"
)

func TestFileWriterConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *output.FileWriterConfig
		fields  []baker.FieldIndex
		wantErr bool
	}{
		{
			name: "all defaults",
			cfg:  &output.FileWriterConfig{},
		},
		{
			name: "{{.Field0}} and len(output.fields) == 1",
			cfg: &output.FileWriterConfig{
				PathString: "/path/{{.Field0}}/file.gz",
			},
			fields: []baker.FieldIndex{0},
		},
		{
			name: "{{.Field0}} and len(output.fields) > 1",
			cfg: &output.FileWriterConfig{
				PathString: "/path/{{.Field0}}/file.gz",
			},
			fields: []baker.FieldIndex{0, 1},
		},

		// error cases
		{
			name: "{{.Field0}} and len(output.fields) == 0",
			cfg: &output.FileWriterConfig{
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
			_, err := output.NewFileWriter(cfg)
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
				DecodedConfig: &output.FileWriterConfig{
					PathString:     filepath.Join(append([]string{t.TempDir()}, comps...)...),
					RotateInterval: rotate,
				},
			},
		}
		fw, err := output.NewFileWriter(cfg)
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
			name:       "field0-out=zst",
			numRecords: 20,
			wait:       1 * time.Millisecond,
			rotate:     time.Second,
			comps:      []string{"{{.Field0}}-out.csv.zst"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, testFileWriterCompareInOut(tt.numRecords, tt.wait, tt.rotate, tt.comps...))
	}
}

// testFileWriterIntegration builds and run a topology reading from
// /testdata/filewriter/input.csv.log.zst and using the FileWriter output,
// configured with the given pathString (in pathString, "TMPDIR" gets replaced
// at runtime by the test case temporary directory).
//
// Once the topology exits, the content of the created temporary directory is
// listed, and compared with the content of the golden file named after the
// test, i.e "testdata/filewriter/TestName". To update the golden file, run:
//  go test -race -run TestName -update
func testFileWriterIntegrationCheckFiles(t *testing.T, pathString string) {
	// This test uses a randomly generated input CSV file.
	//  schema: pick(AAA|BBB|CCC|DDD), digit(22), first, last, email, state
	//  site: https://www.convertcsv.com/generate-test-data.htm

	toml := `
	[fields]
	names = ["kind", "digits", "first", "last", "email", "state"]

	[input]
	name = "list"
	
	[input.config]
	files = ["./testdata/filewriter/input.csv.log.zst"]
	
	[output]
	fields = ["kind"]
	name = "filewriter"
	procs = 1
	[output.config]
	pathstring = "%s"

	#rotateinterval = "2s" NOT USED
`

	tmpDir := t.TempDir()
	toml = fmt.Sprintf(toml, strings.Replace(pathString, "TMPDIR", tmpDir, -1))

	cfg, err := baker.NewConfigFromToml(strings.NewReader(toml),
		baker.Components{
			Inputs:  []baker.InputDesc{input.ListDesc},
			Outputs: []baker.OutputDesc{output.FileWriterDesc},
		})
	if err != nil {
		t.Fatal(err)
	}
	topo, err := baker.NewTopologyFromConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}

	topo.Start()
	topo.Wait()

	// We decompress files since the content of compressed files is not
	// guaranteed to be determinsitic, however it's lossless so we'll use the
	// decompressed file to control the files the output produced.
	decompressFilesInDir(t, tmpDir)

	buf := &bytes.Buffer{}
	if err := dirtree.Write(buf, tmpDir, dirtree.ModeAll, dirtree.ExcludeRoot); err != nil {
		t.Fatal(err)
	}

	golden := filepath.Join("testdata", "filewriter", t.Name())
	if testutil.UpdateGolden != nil && *testutil.UpdateGolden {
		if err := os.WriteFile(golden, buf.Bytes(), os.ModePerm); err != nil {
			t.Fatalf("can't update golden file: %v", err)
		}
	}

	testutil.DiffWithGolden(t, buf.Bytes(), golden)
}

func TestFileWriterIntegrationField0(t *testing.T) {
	testFileWriterIntegrationCheckFiles(t, filepath.Join("TMPDIR", "{{.Field0}}", "out.csv.zst"))
}

func TestFileWriterIntegrationIndex(t *testing.T) {
	testFileWriterIntegrationCheckFiles(t, filepath.Join("TMPDIR", "{{.Index}}", "subdir", "out.csv.zst"))
}

func TestFileWriterIntegrationRotation(t *testing.T) {
	testFileWriterIntegrationCheckFiles(t, filepath.Join("TMPDIR", "{{.Rotation}}", "out.csv.zst"))
}

// decompressFilesInDir decompresses all compressed (zstd/gzip) files it finds
// under root (recursively), and removes the compressed files in files with the
// same name but without the extension.
func decompressFilesInDir(tb testing.TB, root string) {
	var rm []string // files to delete after a successfull walk

	err := filepath.WalkDir(root, func(fullpath string, dirent fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if dirent.IsDir() {
			return nil
		}

		switch filepath.Ext(fullpath) {
		case ".gz", ".zst", ".zstd":
		default:
			return nil
		}

		inf, err := os.Open(fullpath)
		if err != nil {
			return fmt.Errorf("can't open input file: %v", err)
		}
		defer inf.Close()
		zr, err := zip_agnostic.NewReader(inf)
		if err != nil {
			return fmt.Errorf("can't read input file: %v", err)
		}
		defer zr.Close()

		fout, err := os.Create(strings.TrimSuffix(fullpath, filepath.Ext(fullpath)))
		if err != nil {
			return fmt.Errorf("can't create output file: %v", err)
		}
		defer fout.Close()

		if _, err := io.Copy(fout, zr); err != nil {
			return err
		}
		rm = append(rm, fullpath)
		return nil
	})

	if err != nil {
		tb.Fatalf("decompressFilesInDir: error walking directory: %v", err)
	}

	for _, name := range rm {
		if err := os.Remove(name); err != nil {
			tb.Fatalf("after Walk, can't remove %s: %v", name, err)
		}
	}
}
