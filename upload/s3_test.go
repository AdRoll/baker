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

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/awstesting/unit"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// mockS3Service returns a mocked s3.S3 service which records all operations
// related to Upload S3 API calls.
//
// Once all interactions with the returned service have ended, and not before
// that, ops and params can be accessed. ops and params will hold the list of
// AWS S3 API calls and their parameters. For instance, if ops[0] is "PutObject"
// then params[0] is a *s3.PutObjectInput.
func mockS3Service() (svc *s3.S3, ops *[]string, params *[]interface{}) {
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
	srcDir, rmSrcDir := testutil.TempDir(t)
	defer rmSrcDir()

	paths := make([]string, nfiles)
	for i := range paths {
		paths[i] = filepath.Join(srcDir, fmt.Sprintf("file%d", i))
	}

	for _, path := range paths {
		if err := ioutil.WriteFile(path, []byte("foo"), os.ModePerm); err != nil {
			t.Fatal(err)
		}
	}

	cfg := baker.UploadParams{
		ComponentParams: baker.ComponentParams{
			DecodedConfig: &S3Config{
				SourceBasePath: srcDir,
				StagingPath:    "",
				Region:         "us-west-2",
				Bucket:         "my-bucket",
				Prefix:         "my-prefix",
				Concurrency:    16,
				Interval:       1 * time.Millisecond,
			},
		},
	}
	iu, err := newS3(cfg)
	if err != nil {
		t.Fatalf("NewS3Upload(%+v) = %q", cfg, err)
	}

	// Replace S3Upload.manager with a mocked s3 service.
	s, ops, params := mockS3Service()
	u := iu.(*S3)
	u.uploader = s3manager.NewUploaderWithClient(s)
	u.uploader.Concurrency = 10

	// Fill the uploader channel with 10k files.
	upch := make(chan string, len(paths))
	for _, p := range paths {
		upch <- p
	}
	close(upch)

	// Wait for the uploader to exit.
	u.Run(upch)

	if len(*ops) != nfiles {
		t.Fatalf("S3 operations count = %d, want %d", len(*ops), nfiles)
	}
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
