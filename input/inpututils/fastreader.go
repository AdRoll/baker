//go:build !linux && !darwin

package inpututils

import (
	"compress/gzip"
	"io"
)

func newFastGzReader(r io.Reader) (*gzip.Reader, error) {
	return gzip.NewReader(r)
}
