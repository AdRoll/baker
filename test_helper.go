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

	t.Run("set-twice", func(t *testing.T) {
		r := create()
		r.Set(0, []byte("stuff"))
		r.Set(0, []byte("other stuff"))
		got := r.Get(0)
		if !bytes.Equal(got, []byte("other stuff")) {
			t.Errorf("r.Get(0) = %q, want %q", got, "other stuff")
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

	t.Run("Parse+ToText", func(t *testing.T) {
		l := create()
		l.Set(0, []byte("foo"))
		l.Set(1, []byte("bar"))
		l.Set(260, []byte("baz"))

		l.Parse(l.ToText(nil), nil)
		buf1 := l.ToText(nil)

		l.Parse(buf1, nil)
		buf2 := l.ToText(nil)

		if !bytes.Equal(buf1, buf2) {
			t.Errorf("Parse <-> ToText should be idempotent, got:\nbuf1 = %q\nbuf2 = %q", buf1, buf2)
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

	t.Run("copy just parsed + Get", func(t *testing.T) {
		org := create()
		org.Set(0, []byte("foo"))
		org.Set(1, []byte("bar"))
		org.Set(260, []byte("baz"))

		text := org.ToText(nil)

		record := create()
		if err := record.Parse(text, nil); err != nil {
			t.Errorf("Parse error: %v", err)
		}

		cpy := record.Copy()

		if !bytes.Equal(cpy.Get(0), []byte("foo")) {
			t.Errorf("got %q, want %q", cpy.Get(0), []byte("foo"))
		}

		if !bytes.Equal(cpy.Get(1), []byte("bar")) {
			t.Errorf("got %q, want %q", cpy.Get(1), []byte("bar"))
		}

		if !bytes.Equal(cpy.Get(260), []byte("baz")) {
			t.Errorf("got %q, want %q", cpy.Get(260), []byte("baz"))
		}
	})

	t.Run("copy just parsed then modified", func(t *testing.T) {
		org := create()
		org.Set(0, []byte("foo"))
		org.Set(1, []byte("bar"))
		org.Set(260, []byte("baz"))

		if err := org.Parse(org.ToText(nil), nil); err != nil {
			t.Errorf("Parse error: %v", err)
		}

		org.Set(10, []byte("after parser"))

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
