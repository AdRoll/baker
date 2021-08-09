package zip_agnostic

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReader(t *testing.T) {
	lorem := filepath.Join("testdata", "lorem.txt")
	want, err := os.ReadFile(lorem)
	if err != nil {
		t.Fatalf("can't read original file: %s", err)
	}

	fnames := []string{
		lorem,
		lorem + ".zst",
		lorem + ".gz",
	}

	for _, fname := range fnames {
		t.Run(fname, func(t *testing.T) {
			f, err := os.Open(fname)
			if err != nil {
				t.Fatal(err)
			}

			r, err := NewReader(f)
			if err != nil {
				t.Fatalf("zagnosticReader returns %v", err)
			}
			got, err := io.ReadAll(r)
			if err != nil {
				t.Fatalf("couldn't read all: %v", err)
			}

			if !bytes.Equal(got, want) {
				bad := filepath.Join(t.TempDir(), filepath.Base(fname))

				t.Errorf("unexpected buffer content, writing it to: %q", bad)
				if err := os.WriteFile(bad, got, 0644); err != nil {
					t.Fatalf("can't write to %q, %v", bad, err)
				}
			}
		})
	}
}

func TestReader1Byte(t *testing.T) {
	r, err := NewReader(strings.NewReader("0"))
	if err != nil {
		t.Fatal(err)
	}
	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}

	if string(b) != "0" {
		t.Errorf("got b = %q, want %q", b, "0")
	}
}
