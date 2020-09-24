package baker

import (
	"bytes"
	"reflect"
	"testing"
)

// RecordConformanceTest is a test helper that verifies the conformance of
// Record implementation with a set of requirements.
func RecordConformanceTest(t *testing.T, create func() Record) {
	t.Helper()

	t.Run("valid-zero-value", func(t *testing.T) {
		r := create()
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
		r := create()
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

	t.Run("copy", func(t *testing.T) {
		org := create()
		org.Set(0, []byte("foo"))
		org.Set(1, []byte("bar"))
		org.Set(260, []byte("baz"))

		cpy := org.Copy()

		want := org.ToText(nil)
		got := cpy.ToText(nil)

		if !bytes.Equal(got, want) {
			t.Errorf("got %q\nwant %q", got, want)
		}
	})

	t.Run("copy just parsed", func(t *testing.T) {
		org := create()
		org.Set(0, []byte("foo"))
		org.Set(1, []byte("bar"))
		org.Set(260, []byte("baz"))

		text := org.ToText(nil)

		// Now parse but do not call Set
		if err := org.Parse(text, nil); err != nil {
			t.Errorf("Parse error: %v", err)
		}

		cpy := org.Copy()

		want := org.ToText(nil)
		got := cpy.ToText(nil)

		if !bytes.Equal(got, want) {
			t.Errorf("got %q\nwant %q", got, want)
		}
	})

	t.Run("copy metadata", func(t *testing.T) {
		org := create()

		orgmd := Metadata{
			"foo": "bar",
			"bar": 27,
			"baz": struct{ A, b int }{2, 7},
		}
		org.Parse(nil, orgmd)

		cpy := org.Copy()

		for k, orgv := range orgmd {
			v, ok := cpy.Meta(k)
			if !(ok && reflect.DeepEqual(v, orgv)) {
				t.Errorf("cpy.Meta(%q) = %+v, want %+v", k, v, orgv)
			}
		}
	})

	// TODO[interface] : add other conformance tests, Meta, Cache, etc.?
}
