package input

import (
	"bytes"
	"compress/gzip"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	log "github.com/sirupsen/logrus"
	zstd "github.com/valyala/gozstd"

	"github.com/AdRoll/baker"
	"github.com/AdRoll/baker/testutil"
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

func TestListS3(t *testing.T) {
	defer testutil.DisableLogging()()

	ch := make(chan *baker.Data)
	defer close(ch)

	matchLen := 0
	var recordCounter int64
	var getObjCounter int64
	numFiles := 3

	go func() {
		for data := range ch {
			if len(data.Bytes) > 0 {
				atomic.AddInt64(&recordCounter, 1)
				matchLen += len(data.Bytes)
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

	numRecords := 10
	var buf []byte
	for i := 0; i < numRecords; i++ {
		ll := randomLogLine()
		buf = append(buf, ll.ToText(nil)...)
		buf = append(buf, '\n')
	}
	bufLen := len(buf)

	dataFn := func(data interface{}) {
		compressedRecord := zstd.Compress(nil, buf)
		lastModified := aws.Time(time.Now())
		switch d := data.(type) {
		case *s3.ListObjectsV2Output:
			d.IsTruncated = aws.Bool(false)
			d.Contents = []*s3.Object{}
			for i := 0; i < numFiles; i++ {
				d.Contents = append(d.Contents, &s3.Object{Key: aws.String(fmt.Sprintf("path/to/file-%d.log.zst", i))})
			}
			// add a file that doesn't match MatchPath
			d.Contents = append(d.Contents, &s3.Object{Key: aws.String("path/to/file3.log.gz")})
		case *s3.HeadObjectOutput:
			d.ContentLength = aws.Int64(int64(len(compressedRecord)))
			d.LastModified = lastModified
		case *s3.GetObjectOutput:
			atomic.AddInt64(&getObjCounter, 1)
			d.ContentLength = aws.Int64(int64(len(compressedRecord)))
			d.LastModified = lastModified
			d.Body = ioutil.NopCloser(bytes.NewReader(compressedRecord))
		}
	}
	svc, _, _ := testutil.MockS3Service(false, &testutil.MockS3{DataFn: dataFn})
	list.(*List).svc = svc

	finished := make(chan bool)
	go func() {
		err = list.Run(ch)
		if err != nil {
			log.Fatalf("unexpected error %v", err)
		}
		finished <- true
		close(finished)
	}()

	select {
	case <-finished:
		v := int(atomic.LoadInt64(&getObjCounter))
		if v != numFiles {
			t.Fatalf("getObjCounter want: %d, got: %d", numFiles, v)
		}

		v = int(atomic.LoadInt64(&recordCounter))
		if v != numFiles {
			t.Fatalf("recordCounter want: %d, got: %d", numFiles, v)
		}

		if bufLen*numFiles != matchLen {
			t.Fatalf("length want: %d, got: %d", bufLen*numFiles, matchLen)
		}
		return
	case <-time.After(1 * time.Second):
		t.Fatal("input timeout")
	}

}
