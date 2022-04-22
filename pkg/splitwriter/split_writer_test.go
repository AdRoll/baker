package splitwriter

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func expectSplits(t *testing.T, dir string, splits map[string]int) {
	t.Helper()

	for fname, size := range splits {
		path := path.Join(dir, fname)
		fi, err := os.Stat(path)
		if err != nil {
			t.Fatalf("got %s, want split %q = %v bytes", err, fname, size)
		}

		if fi.Size() != int64(size) {
			t.Errorf("got split %q = %v bytes, want %v bytes", fname, fi.Size(), size)
		}
	}
}

func TestSplitWriter(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		name             string            // test name
		fname            string            // fname to write
		maxsize, bufsize int               // splitWriter parameters
		data             string            // data to write in fname
		previous         map[string]string // existing splits and their content
		want             map[string]int    // files we want after the close and their size
	}{
		{
			name:    "just line feeds",
			fname:   "just-line-feed",
			maxsize: 3, bufsize: 2,
			data: "\n\n\n\n",
			want: map[string]int{
				"just-line-feed-part-1": 3,
				"just-line-feed-part-2": 1,
			},
		},
		{
			name:    "no line feeds",
			fname:   "no-line-feeds",
			maxsize: 4, bufsize: 2,
			data: "0123456789abcdefghijklmnopqrstuvwxyz",
			want: map[string]int{
				"no-line-feeds": 36,
			},
		},
		{
			name:    "small file",
			fname:   "smallfile",
			maxsize: 12, bufsize: 4,
			data: "test",
			want: map[string]int{
				"smallfile": 4,
			},
		},
		{
			name:    "big file no split",
			fname:   "big-file-no-split",
			maxsize: 12, bufsize: 4,
			data: "01234567890123",
			want: map[string]int{
				"big-file-no-split": 14,
			},
		},
		{
			name:    "big file 1 split",
			fname:   "big-file-1-split",
			maxsize: 12, bufsize: 4,
			data: "012\n34567890123",
			want: map[string]int{
				"big-file-1-split-part-1": 4,
				"big-file-1-split-part-2": 11,
			},
		},
		{
			name:    "big file + extension 2 split",
			fname:   "big-file-ext-2-split.log",
			maxsize: 6, bufsize: 4,
			data: "012\n3456\n7890123",
			want: map[string]int{
				"big-file-ext-2-split-part-1.log": 4,
				"big-file-ext-2-split-part-2.log": 5,
				"big-file-ext-2-split-part-3.log": 7,
			},
		},
		{
			name: "simple append",
			previous: map[string]string{
				"simple-append.log": "previous-content\n",
			},
			fname:   "simple-append.log",
			maxsize: 24, bufsize: 4,
			data: "01\n",
			want: map[string]int{
				"simple-append.log": 20,
			},
		},
		{
			name: "big after append",
			previous: map[string]string{
				"big-after-append": "previous\n",
			},
			fname:   "big-after-append",
			maxsize: 12, bufsize: 4,
			data: "01234\n",
			want: map[string]int{
				"big-after-append-part-1": 9,
				"big-after-append-part-2": 6,
			},
		},
		{
			name: "big after append 2",
			previous: map[string]string{
				"big-after-append-2": "01\n",
			},
			fname:   "big-after-append-2",
			maxsize: 40, bufsize: 4,
			data: "012\n345678901234567890012345678901234567890\n",
			want: map[string]int{
				"big-after-append-2-part-1": 7,
				"big-after-append-2-part-2": 40,
			},
		},
		{
			name: "already split",
			previous: map[string]string{
				"already-split-part-1.log": "012345\n",
				"already-split-part-2.log": "01\n",
			},
			fname:   "already-split.log",
			maxsize: 8, bufsize: 4,
			data: "012\n34567890123456789\n",
			want: map[string]int{
				"already-split-part-1.log": 7,
				"already-split-part-2.log": 7,
				"already-split-part-3.log": 18,
			},
		},
		{
			name:    "one write multiple splits",
			fname:   "1-write-multi-splits.log",
			maxsize: 5, bufsize: 3,
			data: "0\n012\n0123\n0\n",
			want: map[string]int{
				"1-write-multi-splits-part-1.log": 2,
				"1-write-multi-splits-part-2.log": 4,
				"1-write-multi-splits-part-3.log": 5,
				"1-write-multi-splits-part-4.log": 2,
			},
		},
		{
			name: "one write multiple splits and existing splits",
			previous: map[string]string{
				"1-write-multi-splits-existing-splits-part-1.log": "0\n",
				"1-write-multi-splits-existing-splits-part-2.log": "0",
			},
			fname:   "1-write-multi-splits-existing-splits.log",
			maxsize: 5, bufsize: 3,
			data: "12\n0123\n0\n",
			want: map[string]int{
				"1-write-multi-splits-existing-splits-part-1.log": 2,
				"1-write-multi-splits-existing-splits-part-2.log": 4,
				"1-write-multi-splits-existing-splits-part-3.log": 5,
				"1-write-multi-splits-existing-splits-part-4.log": 2,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for f, data := range tt.previous {
				if err := os.WriteFile(filepath.Join(dir, f), []byte(data), 0666); err != nil {
					t.Fatal(err)
				}
			}

			path := filepath.Join(dir, tt.fname)
			w, err := New(path, int64(tt.maxsize), int64(tt.bufsize))
			if err != nil {
				t.Fatal(err)
			}

			if _, err := w.Write([]byte(tt.data)); err != nil {
				t.Fatal(err)
			}
			if err := w.Close(); err != nil {
				t.Fatal(err)
			}

			expectSplits(t, dir, tt.want)
		})
	}
}

func TestSplitWriterErr(t *testing.T) {
	type swConf struct {
		path             string // path to file
		maxsize, bufsize int    // splitWriter parameters
		data             string // data to write in fname
	}
	initSplitWriter := func(t *testing.T, cf swConf) io.WriteCloser {
		w, err := New(cf.path, int64(cf.maxsize), int64(cf.bufsize))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(cf.data)); err != nil {
			t.Fatal(err)
		}
		return w
	}

	t.Run("doFirstSplit f1 error ", func(t *testing.T) {
		dir := t.TempDir()

		cfg := swConf{
			path:    filepath.Join(dir, "big-file-1-split"),
			maxsize: 12,
			bufsize: 4,
			data:    "012\n34567890123",
		}
		w := initSplitWriter(t, cfg)

		// Create the first part whitout write permission
		f1, _ := os.OpenFile(cfg.path+"-part-1", os.O_CREATE, 0444)
		f1.Close()

		if err := w.Close(); err == nil {
			t.Errorf("get nil, want error")
		}

		// Check that all files where closed
		if err := os.RemoveAll(dir); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("doFirstSplit f2 err", func(t *testing.T) {
		dir := t.TempDir()

		cfg := swConf{
			path:    filepath.Join(dir, "big-file-1-split"),
			maxsize: 12,
			bufsize: 4,
			data:    "012\n34567890123",
		}
		w := initSplitWriter(t, cfg)

		// Create the second part whitout write permission
		f2, _ := os.OpenFile(cfg.path+"-part-2", os.O_CREATE, 0444)
		f2.Close()

		if err := w.Close(); err == nil {
			t.Errorf("get nil, want error")
		}

		// Check that all files where closed
		if err := os.RemoveAll(dir); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("doFirstSplit error on remove", func(t *testing.T) {
		if "windows" != runtime.GOOS {
			t.Skip("Cannot produce an error on file remove on a not windows platform")
		}

		dir := t.TempDir()

		cfg := swConf{
			path:    filepath.Join(dir, "big-file-1-split"),
			maxsize: 12,
			bufsize: 4,
			data:    "012\n34567890123",
		}
		w := initSplitWriter(t, cfg)

		// Open the log file to prevent cancellation during w.Close()
		f, _ := os.OpenFile(cfg.path, os.O_CREATE|os.O_RDWR, 0)

		if err := w.Close(); err == nil {
			t.Errorf("get nil, want error")
		}

		// Check that all files where closed
		f.Close()
		if err := os.RemoveAll(dir); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("doNextSplit f error", func(t *testing.T) {
		dir := t.TempDir()

		cfg := swConf{
			path:    filepath.Join(dir, "big-file-ext-2-split"),
			maxsize: 6,
			bufsize: 4,
			data:    "012\n3456\n7890123",
		}
		w := initSplitWriter(t, cfg)

		// Create the third part whitout write permission
		f2, _ := os.OpenFile(cfg.path+"-part-3", os.O_CREATE, 0444)
		f2.Close()

		if err := w.Close(); err == nil {
			t.Errorf("get nil, want error")
		}

		// Check that all files where closed
		if err := os.RemoveAll(dir); err != nil {
			t.Fatal(err)
		}
	})

}

func TestCloseFiles(t *testing.T) {
	tests := []struct {
		name   string   // test name
		fnames []string // fname to create
		close  []int    // indeces of file to close
	}{
		{
			name:   "good 1",
			fnames: []string{"test0"},
			close:  []int{}, // want no error
		},
		{
			name:   "good 2",
			fnames: []string{"test0", "test1"},
		},
		{
			name:   "good 3",
			fnames: []string{"test0", "test1", "test2"},
		},
		{
			name:   "bad 1",
			fnames: []string{"test0", "test1", "test2"},
			close:  []int{0}, // want error
		},
		{
			name:   "bad 2",
			fnames: []string{"test0", "test1", "test2"},
			close:  []int{1},
		},
		{
			name:   "bad 3",
			fnames: []string{"test0", "test1", "test2"},
			close:  []int{2},
		},
		{
			name:   "bad 1,3",
			fnames: []string{"test0", "test1", "test2"},
			close:  []int{1, 2},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()

			files := make([]*os.File, 0, 3)
			for _, fname := range tt.fnames {
				f, _ := os.OpenFile(filepath.Join(dir, fname), os.O_CREATE|os.O_RDWR, 0)
				files = append(files, f)
			}

			for _, i := range tt.close {
				files[i].Close()
			}

			err := closeFiles(files...)

			// Check returned error
			if len(tt.close) == 0 {
				// Want no error
				if err != nil {
					t.Errorf("get %v, want nil", err)
				}
			} else {
				// Want error
				if err == nil {
					t.Errorf("get nil, want error")
				} else {
					// Check we return the first error
					fname := fmt.Sprintf("test%d", tt.close[0])
					if !strings.Contains(err.Error(), fname) {
						t.Errorf("error %v not contains %q", err, fname)
					}
				}
			}

			// Check that all files were closed
			for _, f := range files {
				err := f.Close()
				if err == nil {
					t.Errorf("file %q not close", f.Name())
				}
			}
		})
	}
}

func Test_nextSplit(t *testing.T) {
	tests := []struct {
		fname     string
		want      string
		wantFirst bool
		wantErr   bool
	}{
		{
			fname:   "",
			wantErr: true,
		},
		{
			fname:     "file",
			wantFirst: true,
			want:      "file-part-1",
		},
		{
			fname: "file-part-1",
			want:  "file-part-2",
		},
		{
			fname: "file-part-2.log",
			want:  "file-part-3.log",
		},
		{
			fname: "file.foo-part-2.log",
			want:  "file.foo-part-3.log",
		},
		{
			fname: "file.foo-part-2",
			want:  "file.foo-part-3",
		},
		{
			fname:   "file.foo-part-0",
			wantErr: true,
		},
		{
			fname: "file.foo-part-9",
			want:  "file.foo-part-10",
		},
		{
			fname: "file.foo-part-99",
			want:  "file.foo-part-100",
		},
	}
	for _, tt := range tests {
		t.Run(tt.fname, func(t *testing.T) {
			got, first, err := nextSplit(tt.fname)
			if (err != nil) != tt.wantErr {
				t.Errorf("nextSplit(%q) error = %v, wantErr %v", tt.fname, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("nextSplit(%q) = %q, want %q", tt.fname, got, tt.want)
			}
			if first != tt.wantFirst {
				t.Errorf("nextSplit(%q), got firstSplit = %v, want %v", tt.fname, first, tt.wantFirst)
			}
		})
	}
}
