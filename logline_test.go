package baker

import (
	"bytes"
	"strings"
	"testing"
)

// This tests ensure parse does not crash when it meets a log line
// with too many separators.
func TestLogLineParse_separators(t *testing.T) {
	maxSeparators := int(LogLineNumFields - 1)
	tests := []struct {
		name  string
		nseps int
		reset bool // Whether the line should be zeroed out after parse.
	}{
		{
			name:  "empty",
			nseps: 0,
			reset: true,
		},
		{
			name:  "1-separator",
			nseps: 1,
			reset: false,
		},
		{
			name:  "max-minus-1-separators",
			nseps: maxSeparators - 1,
			reset: false,
		},
		{
			name:  "max-separators",
			nseps: maxSeparators,
			reset: false,
		},
		{
			name:  "more-than-max-separators",
			nseps: maxSeparators + 1,
			reset: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("!!ALERT!! got panic in Parse: %v", r)
				}
			}()

			b := bytes.Buffer{}
			for i := 0; i < tt.nseps; i++ {
				b.WriteByte(DefaultLogLineFieldSeparator)
			}
			ll := LogLine{FieldSeparator: DefaultLogLineFieldSeparator}
			ll.Parse(b.Bytes(), nil)

			if tt.reset && ll.data != nil {
				t.Errorf("log line not zeroed out but it should")
			}
			if !tt.reset && ll.data == nil {
				t.Errorf("log line got zeroed out but it shouldn't")
			}
		})
	}
}

func TestLogLineToTextWithFieldsHigherThan256(t *testing.T) {
	for i := FieldIndex(0); i < LogLineNumFields; i++ {
		ll := LogLine{FieldSeparator: DefaultLogLineFieldSeparator}
		ll.Set(i, []byte("myvalue"))
		if !bytes.Contains(ll.ToText(nil), []byte("myvalue")) {
			t.Fatalf("Field %d: %s not found in ll.ToText()", i, "myvalue")
		}
	}
}

func TestLogLineMeta(t *testing.T) {
	var ll Record
	ll = &LogLine{FieldSeparator: DefaultLogLineFieldSeparator}
	_, ok := ll.Meta("foo")
	if ok {
		t.Errorf("ll.Meta(%q) = _, %v;  want _, false", "foo", ok)
	}

	ll.Parse(nil, Metadata{"foo": 23})
	val, ok := ll.Meta("foo")
	if !ok || val != 23 {
		t.Errorf("ll.Meta(%q) = %v, %v;  want 23, true", "foo", val, ok)
	}
}

func TestLogLineCache(t *testing.T) {
	var ll Record
	ll = &LogLine{FieldSeparator: DefaultLogLineFieldSeparator}

	testCache := func() {
		_, ok := ll.Cache().Get("foo")
		if ok {
			t.Errorf("ll.Cache().Get(%q) = _, %v;  want _, false", "foo", ok)
		}

		ll.Cache().Set("foo", 23)
		val, ok := ll.Cache().Get("foo")
		if !ok || val != 23 {
			t.Errorf("ll.Cache().Get(%q) = %v, %v;  want 23, true", "foo", val, ok)
		}

		ll.Cache().Set("foo", "hello gopher")
		val, ok = ll.Cache().Get("foo")
		if !ok || val != "hello gopher" {
			t.Errorf("ll.Cache().Get(%q) = %v, %v;  want 23, true", "foo", val, ok)
		}

		ll.Cache().Del("foo")
		val, ok = ll.Cache().Get("foo")
		if ok {
			t.Errorf("ll.Cache().Get(%q) = %v, %v;  want _, false", "foo", val, ok)
		}
	}

	testCache()
	ll.Cache().Clear()
	testCache()
}

func TestLogLineRecordConformance(t *testing.T) {
	createLogLine := func() Record {
		return &LogLine{FieldSeparator: DefaultLogLineFieldSeparator}
	}

	RecordConformanceTest(t, createLogLine)
}

func TestLogLineParseCustomSeparator(t *testing.T) {
	t.Run("default comma separator", func(t *testing.T) {
		text := []byte("value1,value2,,value4")
		ll := LogLine{FieldSeparator: ','}
		ll.Parse(text, nil)
		if !bytes.Equal(ll.Get(0), []byte("value1")) {
			t.Fatalf("want: %v, got: %v", "value1", ll.Get(0))
		}
		if !bytes.Equal(ll.Get(1), []byte("value2")) {
			t.Fatalf("want: %v, got: %v", "value2", ll.Get(1))
		}
		if !bytes.Equal(ll.Get(2), []byte("")) {
			t.Fatalf("want: %v, got: %v", "", ll.Get(2))
		}
		if !bytes.Equal(ll.Get(3), []byte("value4")) {
			t.Fatalf("want: %v, got: %v", "value4", ll.Get(3))
		}
	})

	t.Run("custom dot separator", func(t *testing.T) {
		text := []byte("value1.value2..value4")
		ll := LogLine{FieldSeparator: '.'}
		ll.Parse(text, nil)
		if !bytes.Equal(ll.Get(0), []byte("value1")) {
			t.Fatalf("want: %v, got: %v", "value1", ll.Get(0))
		}
		if !bytes.Equal(ll.Get(1), []byte("value2")) {
			t.Fatalf("want: %v, got: %v", "value2", ll.Get(1))
		}
		if !bytes.Equal(ll.Get(2), []byte("")) {
			t.Fatalf("want: %v, got: %v", "", ll.Get(2))
		}
		if !bytes.Equal(ll.Get(3), []byte("value4")) {
			t.Fatalf("want: %v, got: %v", "value4", ll.Get(3))
		}
	})
}

func TestLogLineToText(t *testing.T) {
	t.Run("default comma separator", func(t *testing.T) {
		ll := LogLine{FieldSeparator: ','}
		ll.Set(0, []byte("value1"))
		ll.Set(1, []byte("value2"))
		ll.Set(3, []byte("value4"))
		text := ll.ToText(nil)
		exp := []byte("value1,value2,,value4")
		if !bytes.Equal(text, exp) {
			t.Fatalf("want: %s got: %s", exp, text)
		}
	})

	t.Run("custom dot separator", func(t *testing.T) {
		ll := LogLine{FieldSeparator: '.'}
		ll.Set(0, []byte("value1"))
		ll.Set(1, []byte("value2"))
		ll.Set(3, []byte("value4"))
		text := ll.ToText(nil)
		exp := []byte("value1.value2..value4")
		if !bytes.Equal(text, exp) {
			t.Fatalf("want: %s got: %s", exp, text)
		}
	})

	t.Run("empty logline", func(t *testing.T) {
		ll := LogLine{FieldSeparator: ','}
		b := ll.ToText(nil)
		if !bytes.Equal(b, []byte("")) {
			t.Fatalf("want: '' got: %s", b)
		}
	})

	t.Run("set", func(t *testing.T) {
		want := []byte(",,value2")

		ll := LogLine{FieldSeparator: ','}
		ll.Set(2, []byte("value2"))
		b := ll.ToText(nil)
		if !bytes.Equal(b, want) {
			t.Fatalf("want: %s got: %s", want, b)
		}
	})

	t.Run("parse", func(t *testing.T) {
		want := []byte("value,value,value")

		ll := LogLine{FieldSeparator: ','}
		ll.Parse(want, nil)
		b := ll.ToText(nil)
		if !bytes.Equal(b, want) {
			t.Fatalf("want: %s got: %s", want, b)
		}
	})

	t.Run("parse and set", func(t *testing.T) {
		want := []byte("value2,value,value")
		text := []byte("value,value,value")

		ll := LogLine{FieldSeparator: ','}
		ll.Parse(text, nil)
		ll.Set(0, []byte("value2"))
		b := ll.ToText(nil)
		if !bytes.Equal(b, want) {
			t.Fatalf("want: %s got: %s", want, b)
		}
	})

	t.Run("parse and set 2", func(t *testing.T) {
		want := []byte("value,value,value,,,value2")
		text := []byte("value,value,value")

		ll := LogLine{FieldSeparator: ','}
		ll.Parse(text, nil)
		ll.Set(5, []byte("value2"))
		b := ll.ToText(nil)
		if !bytes.Equal(b, want) {
			t.Fatalf("want: %s got: %s", want, b)
		}
	})

	t.Run("parse max num fields", func(t *testing.T) {
		values := make([]string, 0, LogLineNumFields)
		for i := 0; i < int(LogLineNumFields); i++ {
			values = append(values, "value")
		}
		want := []byte(strings.Join(values, ","))

		ll := LogLine{FieldSeparator: ','}
		ll.Parse(want, nil)
		b := ll.ToText(nil)
		if !bytes.Equal(b, want) {
			t.Fatalf("want: %s got: %s", want, b)

		}
	})

	t.Run("parse max num fields and set", func(t *testing.T) {
		values := make([]string, 0, LogLineNumFields)
		for i := 0; i < int(LogLineNumFields); i++ {
			values = append(values, "value")
		}
		text := []byte(strings.Join(values, ","))
		values[50] = "other"
		want := []byte(strings.Join(values, ","))

		ll := LogLine{FieldSeparator: ','}
		ll.Parse(text, nil)
		ll.Set(50, []byte("other"))
		b := ll.ToText(nil)
		if !bytes.Equal(b, want) {
			t.Fatalf("want: %s got: %s", want, b)

		}
	})

	t.Run("parse max num field - 1 and set last", func(t *testing.T) {
		values := make([]string, 0, LogLineNumFields)
		for i := 0; i < int(LogLineNumFields)-1; i++ {
			values = append(values, "value")
		}
		text := []byte(strings.Join(values, ","))
		values = append(values, "other")
		want := []byte(strings.Join(values, ","))

		ll := LogLine{FieldSeparator: ','}
		ll.Parse(text, nil)
		ll.Set(LogLineNumFields-1, []byte("other"))
		b := ll.ToText(nil)
		if !bytes.Equal(b, want) {
			t.Fatalf("want: %s got: %s", want, b)
		}
	})
}
