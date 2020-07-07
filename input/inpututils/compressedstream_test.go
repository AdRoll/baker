package inpututils

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/AdRoll/baker"
)

func TestGzipStream(t *testing.T) {
	genfile := func(wpipe io.WriteCloser) {
		defer wpipe.Close()
		w := gzip.NewWriter(wpipe)
		defer w.Close()

		for i := 0; i < 30; i++ {
			line := bytes.Repeat([]byte{'a'}, 15*1024)
			line = append(line, '\n')
			w.Write(line)
		}
	}

	lastModified := time.Unix(1234, 5678)
	opener := func(fn string) (io.ReadCloser, int64, time.Time, *url.URL, error) {
		if fn != "pipe" {
			panic(fn)
		}
		rpipe, wpipe := io.Pipe()
		go genfile(wpipe)
		url := &url.URL{
			Scheme: "pipe",
			Path:   "/fake"}
		return rpipe, 0, lastModified, url, nil
	}

	sizer := func(fn string) (int64, error) {
		return 0, nil
	}

	numlines := 0
	data := make(chan *baker.Data)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		for linesData := range data {
			cnt := bytes.Count(linesData.Bytes, []byte{'\n'})
			numlines += cnt

			lastModifiedMeta, ok := linesData.Meta[MetadataLastModified]
			if !ok {
				t.Errorf("last modified metadata is unavailable")
			}

			if lastModifiedMeta != lastModified {
				t.Errorf(
					"invalid last modified in metadata want:%s got:%s",
					lastModified,
					lastModifiedMeta,
				)
			}

			URLMeta, ok := linesData.Meta[MetadataURL]
			if !ok {
				t.Errorf("url metadata is unavailable")
			}
			if URLMeta == nil ||
				URLMeta.(*url.URL).String() != "pipe:///fake" {
				t.Errorf("invalid url in metadata: %#v", URLMeta)
			}
		}
		wg.Done()
	}()

	done := make(chan bool, 1)
	gz := NewCompressedInput(opener, sizer, done)
	gz.SetOutputChannel(data)
	gz.ProcessFile("pipe")
	gz.NoMoreFiles()
	<-done

	close(data)
	wg.Wait()

	if numlines != 30 {
		t.Errorf("invalid num lines, want:30, got:%d", numlines)
	}
}
