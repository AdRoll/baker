package inpututils

import (
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func getCorpus() []byte {
	files, err := filepath.Glob("*.go")
	if err != nil {
		panic(err)
	}

	var data []byte
	for _, fn := range files {
		body, err := os.ReadFile(fn)
		if err != nil {
			panic(err)
		} else {
			data = append(data, body...)
		}
	}

	data = append(data, data...)
	data = append(data, data...)

	return data
}

func TestGzReader(t *testing.T) {
	data := getCorpus()

	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	w.Write(data)
	w.Close()

	r, err := newFastGzReader(&buf)
	defer r.Close()
	if err != nil {
		t.Fatal(err)
	}
	data2, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(data, data2) {
		t.Error("data mismatch after decompression")
	}
}

func TestGzReadErrorEmpty(t *testing.T) {
	var empty bytes.Buffer

	r, err := newFastGzReader(&empty)
	if err != nil {
		// Error detected, OK
		t.Log(err)
		return
	}

	_, err = io.ReadAll(r)
	if err != nil {
		// Error detected, OK
		t.Log(err)
		return
	}

	err = r.Close()
	if err != nil {
		// Error detected, OK
		t.Log(err)
		return
	}

	t.Fatal("error not detected")
}

func TestGzReadErrorInvalid(t *testing.T) {
	wrong := "ciaociaociaociao"

	r, err := newFastGzReader(strings.NewReader(wrong))
	if err != nil {
		// Error detected, OK
		t.Log(err)
		return
	}

	_, err = io.ReadAll(r)
	if err != nil {
		// Error detected, OK
		t.Log(err)
		return
	}

	err = r.Close()
	if err != nil {
		// Error detected, OK
		t.Log(err)
		return
	}

	t.Fatal("error not detected")
}

func BenchmarkGzip(b *testing.B) {
	data := getCorpus()
	data = append(data, data...)
	data = append(data, data...)
	data = append(data, data...)
	data = append(data, data...)
	data = append(data, data...)

	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	w.Write(data)
	w.Close()

	data = buf.Bytes()
	b.SetBytes(int64(len(data)))
	gz, _ := gzip.NewReader(bytes.NewReader(data))

	for i := 0; i < b.N; i++ {
		gz.Reset(bytes.NewReader(data))
		io.Copy(io.Discard, gz)
	}
}

func BenchmarkFastGzip(b *testing.B) {
	data := getCorpus()
	data = append(data, data...)
	data = append(data, data...)
	data = append(data, data...)
	data = append(data, data...)
	data = append(data, data...)

	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	w.Write(data)
	w.Close()

	data = buf.Bytes()
	b.SetBytes(int64(len(data)))
	gz, _ := newFastGzReader(bytes.NewReader(data))

	for i := 0; i < b.N; i++ {
		gz.Reset(bytes.NewReader(data))
		io.Copy(io.Discard, gz)
	}
}
