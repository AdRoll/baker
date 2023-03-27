package output_test

import (
	"bufio"
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/arl/dirtree"
	"github.com/arl/zt"

	"github.com/AdRoll/baker"
	"github.com/AdRoll/baker/input"
	"github.com/AdRoll/baker/input/inputtest"
	"github.com/AdRoll/baker/output"
	"github.com/AdRoll/baker/testutil"
)

func TestFileWriterConfig(t *testing.T) {
	if !testing.Verbose() {
		defer testutil.LessLogging()()
	}

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
		{
			name: "malformed template",
			cfg: &output.FileWriterConfig{
				PathString: "/path/{{.BadPattern}",
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
		tmpDir := t.TempDir()
		cfg := baker.OutputParams{
			Fields: []baker.FieldIndex{1},
			ComponentParams: baker.ComponentParams{
				DecodedConfig: &output.FileWriterConfig{
					PathString:     filepath.Join(append([]string{tmpDir}, comps...)...),
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
		uploaded := make(map[string]struct{})
		for p := range upch {
			if _, ok := uploaded[p]; ok {
				t.Errorf("file uploaded twice: %q", p)
			}
			uploaded[p] = struct{}{}
		}

		if err := <-errc; err != nil {
			t.Fatalf("fw.Run() error: %v", err)
		}

		// Verify that the set of records sent to the output is equal to the set of
		// records present in the file(s) sent to the uploader.
		uploadedRecords := make(map[int]string)

		for p := range uploaded {
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

			zr, err := zt.NewReader(f)
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

		// Obtain a list of all produced files.
		list, err := dirtree.Sprint(tmpDir, dirtree.Type("f"), dirtree.PrintMode(0))
		if err != nil {
			t.Fatalf("can't list output directory: %s", err)
		}
		produced := make(map[string]struct{})
		for _, fname := range strings.Split(strings.TrimSpace(list), "\n") {
			produced[filepath.Join(tmpDir, fname)] = struct{}{}
		}

		// Check all produced files have been uploaded
		if !reflect.DeepEqual(produced, uploaded) {
			t.Errorf("uploaded and produced files do not match:\n\nuploaded:\n%+v\n\nproduced:\n%+v\n", uploaded, produced)
		}
	}
}

func TestFileWriterCompareInOut(t *testing.T) {
	defer testutil.DisableLogging()()

	tests := []struct {
		name       string
		numRecords int
		wait       time.Duration
		rotate     time.Duration
		comps      []string
	}{
		{
			name:       "year-month-rotation-out.gz",
			numRecords: 500,
			wait:       1 * time.Millisecond,
			rotate:     time.Second,
			comps:      []string{"{{.Year}}", "{{.Month}}", "{{.Rotation}}-out.csv.gz"},
		},
		{
			name:       "year-month-out.gz",
			numRecords: 500,
			wait:       0,
			rotate:     time.Second,
			comps:      []string{"{{.Year}}", "{{.Month}}", "out.csv.gz"},
		},
		{
			name:       "year-month-rotation-out.zst",
			numRecords: 500,
			wait:       1 * time.Millisecond,
			rotate:     time.Second,
			comps:      []string{"{{.Year}}", "{{.Month}}", "{{.Rotation}}-out.csv.zst"},
		},
		{
			name:       "year-month-out.zst",
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
			name:       "field0-out.zst",
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
// test, i.e "testdata/filewriter/TestName". Since the listing computes the
// checksum for each file, it's expected that the topology produces a
// determinstic file/directory hierarchy.
//
// To update the golden file, run: go test -race -run TestName -update
func testFileWriterIntegrationDeterministic(t *testing.T, pathString string) {
	const procs = 1
	const rotate = -1

	tmpDir := t.TempDir()
	testFileWriterIntegration(t, tmpDir, pathString, procs, rotate)

	buf := &bytes.Buffer{}
	if err := dirtree.Write(buf, tmpDir, dirtree.ModeAll, dirtree.ExcludeRoot); err != nil {
		t.Fatal(err)
	}

	golden := filepath.Join("testdata", "filewriter", t.Name()+".golden")
	if testutil.UpdateGolden != nil && *testutil.UpdateGolden {
		if err := os.WriteFile(golden, buf.Bytes(), os.ModePerm); err != nil {
			t.Fatalf("can't update golden file: %v", err)
		}
	}

	testutil.DiffWithGolden(t, buf.Bytes(), golden)
	if t.Failed() {
		dirCpy := filepath.Join(os.TempDir(), t.Name()+".golden")
		fmt.Printf("ERROR: copying output directory to %q for investigation\n\n", dirCpy)
		if err := testutil.CopyDirectory(tmpDir, dirCpy); err != nil {
			t.Fatal(err)
		}
	}
}

// testFileWriterIntegrationCheckRecords builds and run a topology reading from
// /testdata/filewriter/input.csv.log.zst and using the FileWriter output,
// configured with the given pathString (in pathString, "TMPDIR" gets replaced
// at runtime by the test case temporary directory).
//
// Once the topology exits, this test checks that all records that have been
// consumed are present in the produced files (compared with
// "testdata/filewriter/input.sorted.golden". testFileWriterIntegrationCheckRecords
// is useful when the filenames and their content is not expected to be
// deterministic.
func testFileWriterIntegrationCheckRecords(t *testing.T, pathString string, procs int, rotate time.Duration) {
	tmpDir := t.TempDir()
	decompressed := testFileWriterIntegration(t, tmpDir, pathString, procs, rotate)

	// Create a buffer with all the records in ascending order, separated by /n
	var records []string
	for _, name := range decompressed {
		f, err := os.Open(name)
		if err != nil {
			t.Fatal(err)
		}
		scan := bufio.NewScanner(f)
		for scan.Scan() {
			records = append(records, scan.Text())
		}
		f.Close()
	}
	sort.Strings(records)

	var out []byte
	for _, rec := range records {
		out = append(out, rec...)
		out = append(out, '\n')
	}

	testutil.DiffWithGolden(t, out, filepath.Join("testdata", "filewriter", "input.sorted.golden"))
	if t.Failed() {
		tmp, err := os.MkdirTemp(os.TempDir(), t.Name())
		if err != nil {
			t.Fatal(err)
		}

		dirCpy := filepath.Join(tmp, "outdir")
		csvCpy := filepath.Join(tmp, "sorted.csv")
		if err := os.Mkdir(dirCpy, 0777); err != nil {
			t.Fatal(err)
		}

		fmt.Printf("ERROR: copying data for failure investigation in %s\n\t- output directory copy: ./outdir\n\t- incorrect sorted buffer: ./sorted.csv\n\n", tmp)
		if err := testutil.CopyDirectory(tmpDir, dirCpy); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(csvCpy, out, os.ModePerm); err != nil {
			t.Fatal(err)
		}
	}
}

func testFileWriterIntegration(t *testing.T, tmpDir, pathString string, procs int, rotate time.Duration) []string {
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
	procs = %d
	[output.config]
	pathstring = %q
	rotateinterval = %q
`
	if !testing.Verbose() {
		defer testutil.LessLogging()()
	}

	toml = fmt.Sprintf(toml, procs, strings.Replace(pathString, "TMPDIR", tmpDir, -1), rotate)

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
	return decompressFilesInDir(t, tmpDir)
}

func TestFileWriterIntegrationField0(t *testing.T) {
	testFileWriterIntegrationDeterministic(t, filepath.Join("TMPDIR", "{{.Field0}}", "out.csv.zst"))
}

func TestFileWriterIntegrationIndex(t *testing.T) {
	testFileWriterIntegrationDeterministic(t, filepath.Join("TMPDIR", "{{.Index}}", "subdir", "out.csv.zst"))
}

func TestFileWriterIntegrationRotationIndex(t *testing.T) {
	testFileWriterIntegrationDeterministic(t, filepath.Join("TMPDIR", "{{.Rotation}}", "out.csv.zst"))
}

func TestFileWriterIntegrationTimestamp(t *testing.T) {
	const procs = 1
	const rotate = -1
	testFileWriterIntegrationCheckRecords(t, filepath.Join("TMPDIR", "{{.Year}}{{.Month}}{{.Day}}{{.Hour}}{{.Minute}}{{.Second}}-out.csv.zst"), procs, rotate)
}
func TestFileWriterIntegrationProcs(t *testing.T) {
	const procs = 8
	const rotate = -1
	testFileWriterIntegrationCheckRecords(t, filepath.Join("TMPDIR", "{{.Index}}-out.csv.zst"), procs, rotate)
}

func TestFileWriterIntegrationProcsFields0(t *testing.T) {
	const procs = 8
	const rotate = -1
	testFileWriterIntegrationCheckRecords(t, filepath.Join("TMPDIR", "{{.Field0}}.{{.Index}}-out.csv.zst"), procs, rotate)
}

func TestFileWriterIntegrationFastRotation(t *testing.T) {
	const procs = 1
	const rotate = 1 * time.Microsecond
	testFileWriterIntegrationCheckRecords(t, filepath.Join("TMPDIR", "{{.Rotation}}-out.csv.zst"), procs, rotate)
}

func TestFileWriterIntegrationUUID(t *testing.T) {
	const procs = 8
	const rotate = -1
	testFileWriterIntegrationCheckRecords(t, filepath.Join("TMPDIR", "{{.UUID}}-out.csv.zst"), procs, rotate)
}

// decompressFilesInDir decompresses all compressed (zstd/gzip) files it finds
// under root (recursively), and removes the compressed files in files with the
// same name but without the extension. decompressFilesInDir returns the
// absolute path of all files containing decompressed data.
func decompressFilesInDir(tb testing.TB, root string) []string {
	var (
		rm           []string // files to delete after a successfull walk
		decompressed []string // files with decompressed data
	)

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
		zr, err := zt.NewReader(inf)
		if err != nil {
			return fmt.Errorf("can't read input file: %v", err)
		}
		defer zr.Close()

		outPath := strings.TrimSuffix(fullpath, filepath.Ext(fullpath))
		fout, err := os.Create(outPath)
		if err != nil {
			return fmt.Errorf("can't create output file: %v", err)
		}
		defer fout.Close()

		if _, err := io.Copy(fout, zr); err != nil {
			return err
		}
		rm = append(rm, fullpath)
		decompressed = append(decompressed, outPath)
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

	return decompressed
}

// countLines returns the number of lines in a file, either compressed (zstd/gz)
// or uncompressed.
func countLines(r io.Reader) (int, error) {
	zr, err := zt.NewReader(r)
	if err != nil {
		return 0, fmt.Errorf("countLines: %w", err)
	}
	defer zr.Close()

	nlines := 0
	scan := bufio.NewScanner(zr)
	for scan.Scan() {
		nlines++
	}
	if scan.Err() != nil {
		return nlines, fmt.Errorf("countLines: %w", scan.Err())
	}
	return nlines, nil
}

// inputCSV contains 2000 CSV lines.
// Data in the following file has been generated from
// https://www.convertcsv.com/generate-test-data.htm with the following
// 'schema': pick(AAA|BBB|CCC|DDD), digit(22), first, last, email, state
//
//go:embed testdata/filewriter/input.csv
var inputCSV string

const inputCSVNumLines = 2000

func TestFileWriterRotateSize(t *testing.T) {
	defer testutil.DisableLogging()()

	tmpDir := t.TempDir()
	toml := `
		[csv]
		field_separator=","

		[fields]
		names = ["kind", "digits", "first", "last", "email", "state"]

		[input]
		name = "channel"

		[output]
		fields = []
		name = "filewriter"
		procs = 1
		[output.config]
		pathstring = %q
		rotatesize = "2MB"
	`
	if !testing.Verbose() {
		defer testutil.LessLogging()()
	}

	toml = fmt.Sprintf(toml, filepath.Join(tmpDir, "out-{{.Rotation}}.zst"))

	cfg, err := baker.NewConfigFromToml(strings.NewReader(toml),
		baker.Components{
			Inputs:  []baker.InputDesc{inputtest.ChannelDesc},
			Outputs: []baker.OutputDesc{output.FileWriterDesc},
		})
	if err != nil {
		t.Fatal(err)
	}
	topo, err := baker.NewTopologyFromConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}

	const bakerDataCount = 1000 // How many baker.Data blobs we'll send

	in := topo.Input.(*inputtest.Channel)
	go func() {
		for i := 0; i < bakerDataCount; i++ {
			*in <- baker.Data{Bytes: []byte(inputCSV)}
		}
		close(*in)
	}()
	topo.Start()
	topo.Wait()

	// Check the sizes of the created files are all within 1% of the setpoint of
	// 2MB. Only the last file may have a smaller dimension. We don't check the
	// number of files since that depends on the compression algorithm and is
	// thus not stable.
	files, err := dirtree.List(tmpDir, dirtree.Type("f"), dirtree.ModeAll)
	if err != nil {
		t.Fatal(err)
	}

	const (
		sizeEpsilon = 0.05
		rotateSize  = 2_000_000
		minSize     = rotateSize * (1 - sizeEpsilon)
		maxSize     = rotateSize * (1 + sizeEpsilon)
	)
	for i, f := range files {
		if i != len(files)-1 && f.Size < minSize || f.Size > maxSize {
			t.Errorf("file %q, size = %v want %f < size < %f", f.RelPath, f.Size, minSize, maxSize)
		}
	}

	// Sum the the number of lines in all files.
	nlines := 0
	for _, f := range files {
		f, err := os.Open(filepath.Join(tmpDir, f.RelPath))
		if err != nil {
			t.Fatal(err)
		}
		n, err := countLines(f)
		if err != nil {
			t.Fatal(err)
		}
		nlines += n
		f.Close()
	}
	if nlines != bakerDataCount*inputCSVNumLines {
		t.Errorf("total lines = %d, want %d", nlines, bakerDataCount*inputCSVNumLines)
	}
}

func TestFileWriterDiscardEmptyFiles(t *testing.T) {
	defer testutil.DisableLogging()()

	tmpDir := t.TempDir()
	toml := `
		[csv]
		field_separator=","

		[fields]
		names = ["kind", "digits", "first", "last", "email", "state"]

		[input]
		name = "channel"

		[output]
		fields = []
		name = "filewriter"
		procs = 1
		[output.config]
		pathstring = %q
		rotateInterval = "10ms"
		discardEmptyFiles = true
	`
	if !testing.Verbose() {
		defer testutil.LessLogging()()
	}

	toml = fmt.Sprintf(toml, filepath.Join(tmpDir, "file-{{.Hour}}-{{.Minute}}-{{.Second}}-{{.Rotation}}.log.gz"))
	cfg, err := baker.NewConfigFromToml(strings.NewReader(toml),
		baker.Components{
			Inputs:  []baker.InputDesc{inputtest.ChannelDesc},
			Outputs: []baker.OutputDesc{output.FileWriterDesc},
		})
	if err != nil {
		t.Fatal(err)
	}
	topo, err := baker.NewTopologyFromConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}

	in := topo.Input.(*inputtest.Channel)
	go func() {
		*in <- baker.Data{Bytes: []byte(";;;;\n")}
		time.Sleep(200 * time.Millisecond)
		*in <- baker.Data{Bytes: []byte(";;;;\n")}
		time.Sleep(200 * time.Millisecond)

		close(*in)
	}()

	topo.Start()
	topo.Wait()

	files, err := dirtree.List(tmpDir, dirtree.Type("f"), dirtree.ModeAll)
	if err != nil {
		t.Fatal(err)
	}

	if len(files) != 2 {
		t.Fatalf("got %d file(s), want 2", len(files))
	}

	for i := range files {
		f, err := os.Open(files[i].Path)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()

		n, err := countLines(f)
		if err != nil {
			t.Fatal(err)
		}

		if n != 1 {
			t.Fatalf("got %d line(s) in %q, want 1", n, files[0].Path)
		}
	}
}
