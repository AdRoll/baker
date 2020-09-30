// Package splitwriter provides a WriteCloser that writes to a file and splits
// it into smaller files when it's closed.
package splitwriter

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type splitWriter struct {
	fname   string
	f       *os.File
	maxsize int64
	bufsize int64
}

// New returns an io.WriteCloser that writes to fname; when it's closed, fname
// will be split in multiple files each of which having at most maxsize bytes.
// Files are split on offsets where a \n is found, but if no \n is found the
// file isn't split.
func New(fname string, maxsize, bufsize int64) (io.WriteCloser, error) {
	// Ensure split buffer is smaller than the split size.
	if maxsize < bufsize {
		return nil, fmt.Errorf("SplitWriter: maxsize < bufsize")
	}

	w := &splitWriter{
		maxsize: maxsize,
		bufsize: bufsize,
	}

	f, err := openSplit(fname)
	if err != nil {
		return nil, fmt.Errorf("SplitWriter: openSplit: %v", err)
	}
	w.f = f
	w.fname = f.Name()
	return w, nil
}

const (
	fflags = os.O_APPEND | os.O_CREATE | os.O_WRONLY
	fmode  = 0644
)

func open(fname string) (*os.File, error) {
	return os.OpenFile(fname, fflags, fmode)
}

// openSplit opens or creates the last split corresponding to fname
// and returns it. The last split can be fname itself.
func openSplit(fname string) (*os.File, error) {
	for {
		next, _, err := nextSplit(fname)
		if err != nil {
			return nil, err
		}

		if !fileExists(next) {
			break
		}

		fname = next
	}

	return open(fname)
}

func (w *splitWriter) Write(p []byte) (n int, err error) {
	return w.f.Write(p)
}

func (w *splitWriter) Close() error {
	if err := w.f.Close(); err != nil {
		return err
	}

	stat, err := os.Stat(w.fname)
	if err != nil {
		return err
	}

	var f *os.File
	orgf := w.f
	cursize := stat.Size()

	for {
		if cursize <= w.maxsize {
			// All good: splitting not needed
			break
		}

		// Split needed, reopen the file and look for the split point.
		f, err = os.OpenFile(w.fname, os.O_RDWR, 0)
		if err != nil {
			return err
		}

		off, err := w.findSplitPoint(f)
		if err != nil {
			return err
		}
		if off == 0 {
			// Nothing to do since f can't be split.
			break
		}

		f, err := doSplit(f, off)
		if err != nil {
			return fmt.Errorf("splitWriter: doSplit: %v", err)
		}

		// State update for next writes
		w.fname = f.Name()
		w.f = f

		// In case the current file still requires splitting.
		cursize -= off
	}

	if orgf != w.f {
		// Current file pointer is not the one we started with so we must close it
		return w.f.Close()
	}

	return nil
}

// findSplitPoint searches for a suitable split point in f, which is the offset
// of the last line feed which is prior to the split size.
// A split offset of zero indicates that f can't be split.
func (w *splitWriter) findSplitPoint(f *os.File) (off int64, err error) {
	buf := make([]byte, w.bufsize)
	cur := w.maxsize - w.bufsize

	// Read f in reverse, starting from the split size, one buffer at a time.
	for {
		if _, err = f.ReadAt(buf, cur); err != nil {
			return 0, err
		}

		lf := 0
		lastlf := -1
		for lf != -1 {
			lf = bytes.IndexByte(buf[lastlf+1:], '\n')
			if lf != -1 {
				lastlf += lf + 1
			}
		}

		if lastlf != -1 {
			// We found a suitable non-zero split offset.
			off = cur + int64(lastlf) + 1
			break
		}

		if cur == 0 {
			return 0, nil
		}

		// Prepare to read a new buffer
		cur -= w.bufsize
		if cur <= 0 {
			cur = 0
		}
	}

	return off, nil
}

func doSplit(f *os.File, off int64) (*os.File, error) {
	// Find the filename of the next part number
	dir, fname := filepath.Split(f.Name())
	next, first, err := nextSplit(fname)
	if err != nil {
		return nil, err
	}

	if first {
		return doFirstSplit(f, dir, fname, next, off)
	}

	return doNextSplit(f, dir, next, off)
}

// doNextSplit splits f at offset off, transfering the data to the next split.
// It returns the next split.
func doNextSplit(f *os.File, dir, next string, off int64) (*os.File, error) {
	// Transfer the excess data to the next split
	nextpath := filepath.Join(dir, next)
	nextf, err := open(nextpath)
	if err != nil {
		return nil, err
	}

	if _, err := f.Seek(off, io.SeekStart); err != nil {
		return nil, err
	}

	if _, err := io.Copy(nextf, f); err != nil {
		return nil, err
	}

	// Conclude the current split
	if err := f.Truncate(off); err != nil {
		return nil, err
	}

	return nextf, f.Close()
}

// doFirstSplit creates the first splits of f, at offset off.
// That is, from 'file' it creates both 'file.part-1' and 'file.part-2' and
// then removes 'file'.
// It returns the last split.
func doFirstSplit(f *os.File, dir, fname, next1 string, off int64) (*os.File, error) {
	next1path := filepath.Join(dir, next1)
	next2, _, _ := nextSplit(next1)
	next2path := filepath.Join(dir, next2)

	// Open the 2 first parts
	f1, err := open(next1path)
	if err != nil {
		return nil, err
	}
	f2, err := open(next2path)
	if err != nil {
		return nil, err
	}

	// Split the original file into 2 parts
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	if _, err := io.CopyN(f1, f, off); err != nil {
		return nil, err
	}

	if _, err := io.Copy(f2, f); err != nil {
		return nil, err
	}

	// Close the 2 parts and remove the original file
	if err := f.Close(); err != nil {
		return nil, err
	}

	if err := f1.Close(); err != nil {
		return nil, err
	}

	return f2, os.Remove(filepath.Join(dir, fname))
}

var splitFnameRx = regexp.MustCompile(`(\S+)-part-(\d+)(.*)`)

// nextSplit returns the filename indicating the next split considering the
// following splitting rules:
//  - the part number is the string such '.part-XXX' where X is made of [0-9] digits.
//  - the part number is the final component of the filename, it can also be
//    the file extension if there is one.
//  - if a path doesn't contain a split following the previous rules, it will
//    be split considering those rules.
//  - filename must not be empty
func nextSplit(fname string) (next string, first bool, err error) {
	if fname == "" {
		err = errors.New("empty filename")
		return
	}

	m := splitFnameRx.FindAllStringSubmatch(fname, -1)
	if m == nil {
		// Create first split
		ext := filepath.Ext(fname)
		fnoext := strings.TrimSuffix(fname, ext)
		return fmt.Sprintf("%s-part-1%s", fnoext, ext), true, nil
	}

	// Find and increment current part number
	nosplit := m[0][1]
	partnum, _ := strconv.Atoi(m[0][2])
	ext := m[0][3]

	if partnum == 0 {
		return "", false, fmt.Errorf("%q: incorrect split", fname)
	}

	return fmt.Sprintf("%s-part-%d%s", nosplit, partnum+1, ext), false, nil
}

// fileExists reports whether fname exists and is a regular file.
func fileExists(fname string) bool {
	info, err := os.Stat(fname)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
