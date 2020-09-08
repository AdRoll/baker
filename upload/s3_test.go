package upload

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/AdRoll/baker"
	"github.com/AdRoll/baker/testutil"
	log "github.com/sirupsen/logrus"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/awstesting/unit"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// mockS3Service(false) returns a mocked s3.S3 service which records all operations
// related to Upload S3 API calls.
//
// Once all interactions with the returned service have ended, and not before
// that, ops and params can be accessed. ops and params will hold the list of
// AWS S3 API calls and their parameters. For instance, if ops[0] is "PutObject"
// then params[0] is a *s3.PutObjectInput.
func mockS3Service(wantErr bool) (svc *s3.S3, ops *[]string, params *[]interface{}) {
	const respMsg = `<?xml version="1.0" encoding="UTF-8"?>
	<CompleteMultipartUploadOutput>
	   <Location>mockValue</Location>
	   <Bucket>mockValue</Bucket>
	   <Key>mockValue</Key>
	   <ETag>mockValue</ETag>
	</CompleteMultipartUploadOutput>`

	var m sync.Mutex

	ops = &[]string{}
	params = &[]interface{}{}

	partNum := 0
	svc = s3.New(unit.Session)
	svc.Handlers.Unmarshal.Clear()
	svc.Handlers.UnmarshalMeta.Clear()
	svc.Handlers.UnmarshalError.Clear()
	svc.Handlers.Send.Clear()
	svc.Handlers.Send.PushBack(func(r *request.Request) {
		m.Lock()
		defer m.Unlock()

		*ops = append(*ops, r.Operation.Name)
		*params = append(*params, r.Params)

		if wantErr {
			r.HTTPResponse = &http.Response{
				StatusCode: 400,
			}
			return
		}

		r.HTTPResponse = &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte(respMsg))),
		}

		switch data := r.Data.(type) {
		case *s3.CreateMultipartUploadOutput:
			data.UploadId = aws.String("UPLOAD-ID")
		case *s3.UploadPartOutput:
			partNum++
			data.ETag = aws.String(fmt.Sprintf("ETAG%d", partNum))
		case *s3.CompleteMultipartUploadOutput:
			data.Location = aws.String("https://location")
			data.VersionId = aws.String("VERSION-ID")
		case *s3.PutObjectOutput:
			data.VersionId = aws.String("VERSION-ID")
		}
	})

	return svc, ops, params
}

// prepareUploadS3TestFolder creates a temp forlder and the selected number of files in it
func prepareUploadS3TestFolder(t *testing.T, numFiles int) (string, []string, func()) {
	t.Helper()

	// Create a folder to store files to be uploaded
	srcDir, rmSrcDir := testutil.TempDir(t)

	// Write a bunch of files
	var fnames []string
	for i := 0; i < numFiles; i++ {
		fname := filepath.Join(srcDir, fmt.Sprintf("test_file_%d", i))

		if err := ioutil.WriteFile(fname, []byte("abc"), 0644); err != nil {
			t.Fatalf("can't create temp file: %v", err)
		}

		fnames = append(fnames, fname)
	}

	return srcDir, fnames, rmSrcDir
}

func TestS3Upload(t *testing.T) {
	// Through the use of a mocked S3 service, this test verifies that sending
	// 10000 files to an S3Upload results in 10000 S3 Upload API calls.
	// It's always important to run tests with the race detector enabled but
	// specially this one since there's a lot of concurrency involved.
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	defer testutil.DisableLogging()()

	// Create many files.
	const nfiles = 10000
	srcDir, paths, rmSrcDir := prepareUploadS3TestFolder(t, nfiles)
	defer rmSrcDir()

	cfg := baker.UploadParams{
		ComponentParams: baker.ComponentParams{
			DecodedConfig: &S3Config{
				SourceBasePath: srcDir,
				StagingPath:    "",
				Region:         "us-west-2",
				Bucket:         "my-bucket",
				Prefix:         "my-prefix",
				Interval:       1 * time.Millisecond,
			},
		},
	}
	iu, err := newS3(cfg)
	if err != nil {
		t.Fatalf("NewS3Upload(%+v) = %q", cfg, err)
	}

	// Replace S3Upload.manager with a mocked s3 service.
	s, ops, params := mockS3Service(false)
	u := iu.(*S3)
	u.uploader = s3manager.NewUploaderWithClient(s)

	// Fill the uploader channel with 10k files.
	upch := make(chan string, len(paths))
	for _, p := range paths {
		upch <- p
	}
	close(upch)

	// Wait for the uploader to exit.
	u.Run(upch)

	if len(*ops) != nfiles {
		t.Fatalf("S3 operation params count = %d, want %d", len(*ops), nfiles)
	}

	// Check all operations are PutObject.
	for i := range *ops {
		if (*ops)[i] != "PutObject" {
			t.Fatalf("ops[%d] = %q, want PutObject", i, (*ops)[i])
		}
	}

	// Check parameters to PutObject are what we expect.
	type stringset map[string]struct{}
	fnames := make(stringset)

	for i := range *params {
		putObj, ok := (*params)[i].(*s3.PutObjectInput)
		if !ok {
			t.Fatalf("type of params[i] = %T [%+v], want *s3.PutObjectInput", putObj, putObj)
		}
		if *putObj.Bucket != "my-bucket" {
			t.Errorf("params[%d].Bucket = %q, want %q", i, *putObj.Bucket, "my-bucket")
		}
		if !strings.HasPrefix(*putObj.Key, "my-prefix/") {
			t.Errorf("params[%d].Key = %q, want prefix = %q", i, *putObj.Key, "my-prefix/")
		}
		fnames[*putObj.Key] = struct{}{}
	}

	if len(fnames) != nfiles {
		t.Errorf("Wrong number of unique filename: %d, want %d", len(fnames), nfiles)
	}
}

func Test_uploadDirectory(t *testing.T) {
	defer testutil.DisableLogging()()
	// Create a folder to store files to be uploaded
	numFiles := 10
	srcDir, _, rmSrcDir := prepareUploadS3TestFolder(t, numFiles)
	defer rmSrcDir()

	cfg := baker.UploadParams{
		ComponentParams: baker.ComponentParams{
			DecodedConfig: &S3Config{
				SourceBasePath: srcDir,
				StagingPath:    srcDir,
				Bucket:         "my-bucket",
				Concurrency:    5,
				Retries:        3,
			},
		},
	}
	iu, err := newS3(cfg)
	if err != nil {
		t.Fatalf("NewS3Upload(%+v) = %q", cfg, err)
	}
	s, ops, _ := mockS3Service(false)
	u := iu.(*S3)
	u.uploader = s3manager.NewUploaderWithClient(s)
	u.uploader.Concurrency = 5

	if err := u.uploadDirectory(); err != nil {
		log.Fatal(err)
	}
	if len(*ops) != numFiles {
		t.Fatalf("S3 operations count = %d, want %d", len(*ops), numFiles)
	}

	for i := range *ops {
		if (*ops)[i] != "PutObject" {
			t.Fatalf("ops[%d] = %q, want PutObject", i, (*ops)[i])
		}
	}
}

func Test_uploadDirectoryError(t *testing.T) {
	defer testutil.DisableLogging()()

	numFiles := 10
	srcDir, _, rmSrcDir := prepareUploadS3TestFolder(t, numFiles)
	defer rmSrcDir()

	t.Run("ExitOnError: false", func(t *testing.T) {
		cfg := baker.UploadParams{
			ComponentParams: baker.ComponentParams{
				DecodedConfig: &S3Config{
					SourceBasePath: srcDir,
					StagingPath:    srcDir,
					Bucket:         "my-bucket",
					Concurrency:    5,
					Retries:        3,
					ExitOnError:    false,
				},
			},
		}
		iu, err := newS3(cfg)
		if err != nil {
			t.Fatalf("NewS3Upload(%+v) = %q", cfg, err)
		}
		s, _, _ := mockS3Service(true)
		u := iu.(*S3)
		u.uploader = s3manager.NewUploaderWithClient(s)

		if err := u.uploadDirectory(); err != nil {
			log.Fatal(err)
		}
		if int(u.totaln) != 0 {
			t.Fatalf("uploaded: want: %d, got: %d", 0, int(u.totaln))
		}

		if int(u.totalerr) != numFiles*u.Cfg.Retries {
			t.Fatalf("errors: want: %d, got: %d", numFiles*u.Cfg.Retries, int(u.totalerr))
		}
	})

	t.Run("ExitOnError: true", func(t *testing.T) {
		cfg := baker.UploadParams{
			ComponentParams: baker.ComponentParams{
				DecodedConfig: &S3Config{
					SourceBasePath: srcDir,
					StagingPath:    srcDir,
					Bucket:         "my-bucket",
					Concurrency:    5,
					Retries:        3,
					ExitOnError:    true,
				},
			},
		}
		iu, err := newS3(cfg)
		if err != nil {
			t.Fatalf("NewS3Upload(%+v) = %q", cfg, err)
		}
		s, _, _ := mockS3Service(true)
		u := iu.(*S3)
		u.uploader = s3manager.NewUploaderWithClient(s)

		if err := u.uploadDirectory(); err == nil {
			t.Fatalf("expected error")
		}

		// Uploads run parallelized so we can't expect that only 1 error will happen
		// before returning, but for sure they can't be more than the number of concurrency
		if int(u.totalerr) > u.Cfg.Concurrency {
			t.Fatalf("errors: want: <=%d, got: %d", u.Cfg.Concurrency, int(u.totalerr))
		}
	})
}

func TestRun(t *testing.T) {
	defer testutil.DisableLogging()()

	tmpDir, fnames, rmTmpDir := prepareUploadS3TestFolder(t, 1)
	defer rmTmpDir()
	fname := fnames[0]

	stagingDir, err := ioutil.TempDir("", t.Name())
	if err != nil {
		t.Fatalf("Can't setup test: %v", err)
	}
	defer os.RemoveAll(stagingDir)

	cfg := baker.UploadParams{
		ComponentParams: baker.ComponentParams{
			DecodedConfig: &S3Config{
				SourceBasePath: stagingDir,
				StagingPath:    tmpDir,
				Bucket:         "my-bucket",
				Concurrency:    5,
				Retries:        3,
			},
		},
	}
	iu, err := newS3(cfg)
	if err != nil {
		t.Fatalf("NewS3Upload(%+v) = %q", cfg, err)
	}
	s, _, _ := mockS3Service(false)
	u := iu.(*S3)
	u.uploader = s3manager.NewUploaderWithClient(s)

	upCh := make(chan string)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := u.Run(upCh); err != nil {
			t.Fatal(err)
		}
	}()

	upCh <- fname
	u.Stop()
	wg.Wait()

	if int(u.totalerr) != 0 {
		t.Fatalf("totalerr: want: %d, got: %d", 0, int(u.totalerr))
	}

	if int(u.totaln) != 1 {
		t.Fatalf("totaln: want: %d, got: %d", 1, int(u.totaln))
	}
}

func TestRunExitOnError(t *testing.T) {
	defer testutil.DisableLogging()()

	tmpDir, fnames, rmTmpDir := prepareUploadS3TestFolder(t, 1)
	defer rmTmpDir()
	fname := fnames[0]

	stagingDir, err := ioutil.TempDir("", t.Name())
	if err != nil {
		t.Fatalf("Can't setup test: %v", err)
	}
	defer os.RemoveAll(stagingDir)

	cfg := baker.UploadParams{
		ComponentParams: baker.ComponentParams{
			DecodedConfig: &S3Config{
				SourceBasePath: stagingDir,
				StagingPath:    tmpDir,
				Bucket:         "my-bucket",
				ExitOnError:    true,
			},
		},
	}
	iu, err := newS3(cfg)
	if err != nil {
		t.Fatalf("NewS3Upload(%+v) = %q", cfg, err)
	}
	s, _, _ := mockS3Service(true)
	u := iu.(*S3)
	u.uploader = s3manager.NewUploaderWithClient(s)

	upCh := make(chan string)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := u.Run(upCh); err != nil {
			t.Fatal(err)
		}
	}()

	upCh <- fname
	u.Stop()
	wg.Wait()

	if int(u.totalerr) != 1 {
		t.Fatalf("totalerr: want: %d, got: %d", 1, int(u.totalerr))
	}

	if int(u.totaln) != 1 {
		t.Fatalf("totaln: want: %d, got: %d", 1, int(u.totaln))
	}
}

func TestRunNotExitOnError(t *testing.T) {
	defer testutil.DisableLogging()()

	tmpDir, fnames, rmTmpDir := prepareUploadS3TestFolder(t, 1)
	defer rmTmpDir()
	fname := fnames[0]

	stagingDir, err := ioutil.TempDir("", t.Name())
	if err != nil {
		t.Fatalf("Can't setup test: %v", err)
	}
	defer os.RemoveAll(stagingDir)

	cfg := baker.UploadParams{
		ComponentParams: baker.ComponentParams{
			DecodedConfig: &S3Config{
				SourceBasePath: stagingDir,
				StagingPath:    tmpDir,
				Bucket:         "my-bucket",
				Concurrency:    5,
				Retries:        3,
				ExitOnError:    false,
			},
		},
	}
	iu, err := newS3(cfg)
	if err != nil {
		t.Fatalf("NewS3Upload(%+v) = %q", cfg, err)
	}
	s, _, _ := mockS3Service(true)
	u := iu.(*S3)
	u.uploader = s3manager.NewUploaderWithClient(s)

	upCh := make(chan string)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := u.Run(upCh); err != nil {
			t.Fatal(err)
		}
	}()

	upCh <- fname
	u.Stop()
	wg.Wait()

	if int(u.totalerr) > 1*u.Cfg.Retries {
		t.Fatalf("totalerr: want: <=%d, got: %d", 1*u.Cfg.Retries, int(u.totalerr))
	}

	if int(u.totaln) != 1 {
		t.Fatalf("totaln: want: %d, got: %d", 1, int(u.totaln))
	}
}

func Test_move(t *testing.T) {
	srcDir, err := ioutil.TempDir("", t.Name())
	if err != nil {
		t.Fatalf("Can't setup test: %v", err)
	}
	defer os.RemoveAll(srcDir)

	trgtDir, err := ioutil.TempDir("", fmt.Sprintf("%s-trgt", t.Name()))
	if err != nil {
		t.Fatalf("Can't setup test: %v", err)
	}
	defer os.RemoveAll(trgtDir)

	srcFile := filepath.Join(srcDir, "test_file")
	trgtFile := filepath.Join(trgtDir, "test_file")

	if err := ioutil.WriteFile(srcFile, []byte("abc"), 0644); err != nil {
		t.Fatalf("can't create temp file: %v", err)
	}

	s3 := &S3{
		Cfg: &S3Config{
			StagingPath:    trgtDir,
			SourceBasePath: srcDir,
		},
	}

	if err := s3.move(srcFile); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(trgtFile); err != nil {
		t.Error("moved file not found")
	}

	if _, err := os.Stat(srcFile); err == nil {
		t.Error("source file still there")
	}
}
