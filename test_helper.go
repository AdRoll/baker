package baker

import (
	"bytes"
	"testing"
)

// RecordConformanceTest is a test helper that verifies the conformance of
// Record implementation with a set of requirements.
func RecordConformanceTest(t *testing.T, r Record) {
	t.Helper()

	t.Run("valid-zero-value", func(t *testing.T) {
		r.Set(0, []byte("stuff"))
		r.Set(1, []byte("other stuff"))
		got := r.Get(0)
		if !bytes.Equal(got, []byte("stuff")) {
			t.Errorf("r.Get(0) = %q, want %q", got, "stuff")
		}

		got = r.Get(1)
		if !bytes.Equal(got, []byte("other stuff")) {
			t.Errorf("r.Get(1) = %q, want %q", got, "other stuff")
		}
	})

	t.Run("valid-after-clear", func(t *testing.T) {
		r.Set(0, []byte("stuff"))
		r.Clear()
		got := r.Get(0)
		if got != nil {
			t.Errorf("r.Get(0) = %q, want nil", got)
		}

		r.Set(0, []byte("stuff"))
		got = r.Get(0)
		if !bytes.Equal(got, []byte("stuff")) {
			t.Errorf("r.Get(0) = %q, want %q", got, "stuff")
		}
	})

	// TODO[interface] : add other conformance tests, Meta, Cache, etc.?
}
