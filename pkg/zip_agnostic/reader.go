package zip_agnostic

import (
	"bytes"
	"fmt"
	"io"

	"github.com/klauspost/compress/gzip"
	"github.com/valyala/gozstd"
)

const (
	gzipHeader = "\x1f\x8b"
	zstdHeader = "\x28\xb5\x2f\xfd"
)

// NewReader returns an io.ReadCloser that reads from r, whether r is a reader
// over compressed data or not. It supports gzip or zst.
//
// Note: NewReader is an utility function provided as a best effort, it's still
// possible to trick it into thinking a reader contains zstd or gzip compressed
// data, while in fact it's not.
func NewReader(r io.Reader) (io.ReadCloser, error) {
	var hdr [4]byte
	n, err := r.Read(hdr[:])
	switch {
	case err == io.EOF || n < len(hdr):
		return io.NopCloser(bytes.NewReader(hdr[:n])), nil
	case err != nil:
		return nil, fmt.Errorf("zip_agnostic: can't read: %v", err)
	}

	zr := &reader{r: io.NopCloser(r), hdr: hdr[:]}

	if bytes.Equal(hdr[:2], []byte(gzipHeader)) {
		r, err := gzip.NewReader(zr)
		if err != nil {
			return nil, fmt.Errorf("zip_agnostic (gzip): can't read: %v", err)
		}
		return r, nil
	}

	if bytes.Equal(hdr[:4], []byte(zstdHeader)) {
		zstdr := gozstd.NewReader(zr)
		return makeReadCloser(zstdr, func() error { zstdr.Release(); return nil }), nil
	}

	return io.NopCloser(zr), nil
}

type reader struct {
	r    io.ReadCloser
	hdr  []byte
	hoff int // read header bytes
}

func (r *reader) Read(p []byte) (n int, err error) {
	if r.hdr != nil {
		n = copy(p, r.hdr[r.hoff:])
		if n == len(r.hdr) {
			// The whole header has been read, forward subsequent
			// calls to the wrapped reader.
			r.hdr = nil
			return n, nil
		}
	}

	return r.r.Read(p)
}

// makeReadCloser converts an io.Reader and a close function into a ReadCloser.
func makeReadCloser(r io.Reader, close func() error) io.ReadCloser {
	return &readCloser{Reader: r, close: close}
}

type readCloser struct {
	io.Reader
	close func() error
}

func (rc *readCloser) Close() error {
	err := rc.close()
	if err != nil {
		return fmt.Errorf("zip_agnostic: close: %v", err)
	}
	return nil
}
