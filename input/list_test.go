package input

import (
	"bytes"
	"compress/gzip"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	log "github.com/sirupsen/logrus"
	zstd "github.com/valyala/gozstd"

	"github.com/AdRoll/baker"
	"github.com/AdRoll/baker/testutil"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/awstesting/unit"
	"github.com/aws/aws-sdk-go/service/s3"
)

func randomLogLine() baker.Record {
	var ll baker.LogLine

	rand := rand.New(rand.NewSource(0))
	t0 := time.Date(2015, 8, 1, 15, 0, 0, 0, time.UTC)
	tlen := 15 * 24 * time.Hour
	types := []string{"ty1", "ty2", "ty3", "ty4"}

	// Cookie
	var cookie [16]byte
	rand.Read(cookie[:])
	ll.Set(22, []byte(hex.EncodeToString(cookie[:])))

	// Timestamp
	ts := t0.Add(time.Duration(rand.Int63n(int64(tlen))))
	ll.Set(19, []byte(strconv.Itoa(int(ts.Unix()))))

	// Type
	typ := types[rand.Intn(len(types))]
	ll.Set(0, []byte(typ))

	return &ll
}

func makeTestLog(tb testing.TB, dir, fn string, numlines int) string {
	tb.Helper()

	fn = dir + "/" + fn

	f, err := os.Create(fn)
	if err != nil {
		tb.Fatalf("can't create test log file: %v", err)
	}
	defer f.Close()

	gzf := gzip.NewWriter(f)
	defer gzf.Close()

	for i := 0; i < numlines; i++ {
		ll := randomLogLine()
		buf := ll.ToText(nil)
		buf = append(buf, '\n')
		gzf.Write(buf)
	}

	return fn
}

func TestListBasic(t *testing.T) {
	dir, rmdir := testutil.TempDir(t)
	defer rmdir()

	makeTestLog(t, dir, "test7.log.gz", 7)
	makeTestLog(t, dir, "test100.log.gz", 100)
	makeTestLog(t, dir, "test500.log.gz", 500)
	makeTestLog(t, dir, "test1233.log.gz", 1233)
	ioutil.WriteFile(dir+"/"+"testlist600",
		[]byte(dir+"/"+"test100.log.gz"+"\n"+dir+"/"+"test500.log.gz"+"\n"),
		0777)
	ioutil.WriteFile(dir+"/"+"buglist",
		[]byte(dir+"/"+"test100.log.gz"+"\n"+dir+"/"+"nonesisting.log.gz"+"\n"),
		0777)

	var tests = []struct {
		Files []string
		Lines int64
	}{
		{[]string{"test7.log.gz"}, 7},
		{[]string{"test1233.log.gz"}, 1233},
		{[]string{"test100.log.gz", "test500.log.gz"}, 600},
		{[]string{"@testlist600"}, 600},
		{[]string{"test100.log.gz", "@testlist600", "test7.log.gz"}, 707},
		{[]string{"test100.log.gz", "nonexisting.log.gz"}, -1},
		{[]string{"test100.log.gz", "@buglist"}, -1},
		{[]string{"@-", "@buglist"}, -1},
	}

	ch := make(chan *baker.Data)
	defer close(ch)

	var counter int64
	go func() {
		for data := range ch {
			atomic.AddInt64(&counter, int64(bytes.Count(data.Bytes, []byte{'\n'})))
			// Check the metadata contains a valid last modified date
			if v := data.Meta["last_modified"]; !v.(time.Time).After(time.Unix(0, 0)) {
				t.Errorf("Invalid last modified time in file, it should be after %s.", time.Unix(0, 0))
			}
		}
	}()

	for _, test := range tests {
		for idx := range test.Files {
			if test.Files[idx][0] == '@' {
				test.Files[idx] = "@" + dir + "/" + test.Files[idx][1:]
			} else {
				test.Files[idx] = dir + "/" + test.Files[idx]
			}
		}
		cfg := baker.InputParams{
			ComponentParams: baker.ComponentParams{
				DecodedConfig: &ListConfig{Files: test.Files},
			},
		}
		list, err := NewList(cfg)
		if err != nil {
			t.Error("Error creating List:", err)
			continue
		}

		atomic.StoreInt64(&counter, 0)
		err = list.Run(ch)

		if test.Lines == -1 {
			if err == nil {
				t.Errorf("expected error but not found, files %v", test.Files)
			}
		} else {
			if err != nil {
				t.Error("Error running List:", err)
				continue
			}

			c := atomic.LoadInt64(&counter)
			if c != test.Lines {
				t.Errorf("Invalid number of lines: exp=%d, got=%d", test.Lines, c)
			}
		}
	}
}

func TestListInvalidStdin(t *testing.T) {

	piper, pipew, err := os.Pipe()
	if err != nil {
		t.Error(err)
		return
	}

	stdin = piper
	defer func() {
		stdin = os.Stdin
	}()

	ch := make(chan *baker.Data)
	defer close(ch)

	var counter int64
	go func() {
		for data := range ch {
			atomic.AddInt64(&counter, int64(bytes.Count(data.Bytes, []byte{'\n'})))
			// Check the metadata contains a valid last modified date
			if v := data.Meta["last_modified"]; !v.(time.Time).After(time.Unix(0, 0)) {
				t.Errorf("Invalid last modified time in file, it should be after %s.", time.Unix(0, 0))
			}
		}
	}()

	cfg := baker.InputParams{
		ComponentParams: baker.ComponentParams{
			DecodedConfig: &ListConfig{Files: []string{"@-"}},
		},
	}
	list, err := NewList(cfg)
	if err != nil {
		t.Error("Error creating List:", err)
		return
	}

	finished := make(chan bool)

	// Write an invalid file name, and then keep the pipe open and pending.
	// This simulates an invalid file on stdin before stdin is closed, which
	// should trigger an immediate abort of List, without waiting for stdin
	// to be closed.
	go pipew.WriteString("invalidfile.tar.gz\n")
	go func() {
		err = list.Run(ch)
		if err == nil {
			t.Error("expected error but nil returned")
		}
		finished <- true
		close(finished)
	}()

	select {
	case <-finished:
		return
	case <-time.After(1 * time.Second):
		t.Error("input timeout")
	}
}

func TestListS3Folder(t *testing.T) {
	defer testutil.DisableLogging()()

	ch := make(chan *baker.Data)

	generatedFiles := 3
	generatedRecords := 10
	receivedBytesLen := 0
	var receivedFilesContent int64

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for data := range ch {
			if len(data.Bytes) > 0 {
				atomic.AddInt64(&receivedFilesContent, 1)
				receivedBytesLen += len(data.Bytes)
			}
		}
	}()

	cfg := baker.InputParams{
		ComponentParams: baker.ComponentParams{
			DecodedConfig: &ListConfig{
				Files:     []string{"@s3://bucket-name/path-prefix/"},
				MatchPath: ".*\\.log\\.zst",
			},
		},
	}

	list, err := NewList(cfg)
	if err != nil {
		t.Error("Error creating List:", err)
		return
	}

	svc, recordsLen, getObjCounter := mockS3Service(t, generatedFiles, generatedRecords, false)
	list.(*List).svc = svc

	if err := list.Run(ch); err != nil {
		log.Fatalf("unexpected error %v", err)
	}
	close(ch)
	wg.Wait()

	if int(*getObjCounter) != generatedFiles {
		t.Errorf("getObjCounter want: %d, got: %d", generatedFiles, int(*getObjCounter))
	}

	if int(receivedFilesContent) != generatedFiles {
		t.Fatalf("receivedFilesContent want: %d, got: %d", generatedFiles, int(receivedFilesContent))
	}

	if recordsLen*generatedFiles != receivedBytesLen {
		t.Fatalf("length want: %d, got: %d", recordsLen*generatedFiles, receivedBytesLen)
	}
}

func TestListS3Manifest(t *testing.T) {
	defer testutil.DisableLogging()()

	ch := make(chan *baker.Data)

	var receivedFilesContent int64

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for data := range ch {
			if len(data.Bytes) > 0 {
				atomic.AddInt64(&receivedFilesContent, 1)
			}
		}
	}()

	cfg := baker.InputParams{
		ComponentParams: baker.ComponentParams{
			DecodedConfig: &ListConfig{
				Files:     []string{"@s3://bucket-name/path-prefix/manifest"},
				MatchPath: ".*\\.log\\.zst",
			},
		},
	}

	list, err := NewList(cfg)
	if err != nil {
		t.Error("Error creating List:", err)
		return
	}

	svc, _, getObjCounter := mockS3Service(t, 0, 2, true)
	list.(*List).svc = svc

	if err := list.Run(ch); err != nil {
		log.Fatalf("unexpected error %v", err)
	}
	close(ch)
	wg.Wait()

	if int(*getObjCounter) != 3 { // 1 manifest + 2 files into it
		t.Errorf("getObjCounter want: %d, got: %d", 3, int(*getObjCounter))
	}

	if int(receivedFilesContent) != 2 {
		t.Fatalf("receivedFilesContent want: %d, got: %d", 2, int(receivedFilesContent))
	}
}

func mockS3Service(t *testing.T, generatedFiles, generatedRecords int, getManifest bool) (*s3.S3, int, *int64) {
	t.Helper()
	var counter int64
	var buf []byte
	for i := 0; i < generatedRecords; i++ {
		ll := randomLogLine()
		buf = append(buf, ll.ToText(nil)...)
		buf = append(buf, '\n')
	}

	compressedRecord := zstd.Compress(nil, buf)
	lastModified := aws.Time(time.Now())
	var manifestCalled bool

	svc := s3.New(unit.Session)
	svc.Handlers.Unmarshal.Clear()
	svc.Handlers.UnmarshalMeta.Clear()
	svc.Handlers.UnmarshalError.Clear()
	svc.Handlers.Send.Clear()
	var m sync.Mutex
	svc.Handlers.Send.PushBack(func(r *request.Request) {
		m.Lock()
		defer m.Unlock()
		r.HTTPResponse = &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte(""))),
		}

		switch data := r.Data.(type) {
		case *s3.ListObjectsV2Output:
			data.IsTruncated = aws.Bool(false)
			data.Contents = []*s3.Object{}
			for i := 0; i < generatedFiles; i++ {
				data.Contents = append(data.Contents, &s3.Object{Key: aws.String(fmt.Sprintf("path/to/file-%d.log.zst", i))})
			}
			// add a file that doesn't match MatchPath
			data.Contents = append(data.Contents, &s3.Object{Key: aws.String("path/to/file3.log.gz")})

		case *s3.HeadObjectOutput:
			data.ContentLength = aws.Int64(int64(len(compressedRecord)))
			data.LastModified = lastModified

		case *s3.GetObjectOutput:
			atomic.AddInt64(&counter, 1)
			if getManifest && !manifestCalled {
				var manifest []byte
				for i := 0; i < generatedRecords; i++ {
					manifest = append(manifest, []byte(fmt.Sprintf("s3://bucket-name/file-%d-from_manifest.log.zst\n", i))...)
				}
				manifestCalled = true
				data.ContentLength = aws.Int64(int64(len(manifest)))
				data.LastModified = lastModified
				data.Body = ioutil.NopCloser(bytes.NewReader(manifest))
			} else {
				data.ContentLength = aws.Int64(int64(len(compressedRecord)))
				data.LastModified = lastModified
				data.Body = ioutil.NopCloser(bytes.NewReader(compressedRecord))
			}
		}
	})

	return svc, len(buf), &counter
}
