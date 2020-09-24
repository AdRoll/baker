package baker

import (
	"bytes"
	"testing"
)

// This tests ensure parse does not crash when it meets a log line
// with too many separators
func TestLogLineParse_separators(t *testing.T) {
	tests := []struct {
		name  string
		nseps int
		reset bool // whether the line should be zeroed out after parse
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
			nseps: int(LogLineNumFields - 1),
			reset: false,
		},
		{
			name:  "max-separators",
			nseps: int(LogLineNumFields),
			reset: false,
		},
		{
			name:  "more-than-max-separators",
			nseps: int(LogLineNumFields + 1),
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
		ll := LogLine{FieldSeparator: 44}
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
		ll := LogLine{FieldSeparator: 46}
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

func TestLogLineToTextCustomSeparator(t *testing.T) {
	t.Run("default comma separator", func(t *testing.T) {
		ll := LogLine{FieldSeparator: 44}
		ll.Set(0, []byte("value1"))
		ll.Set(1, []byte("value2"))
		ll.Set(3, []byte("value4"))
		text := ll.ToText(nil)
		exp := []byte("value1,value2,,value4,,")
		if !bytes.Equal(text, exp) {
			t.Fatalf("want: %s got: %s", exp, text)
		}
	})

	t.Run("custom dot separator", func(t *testing.T) {
		ll := LogLine{FieldSeparator: 46}
		ll.Set(0, []byte("value1"))
		ll.Set(1, []byte("value2"))
		ll.Set(3, []byte("value4"))
		text := ll.ToText(nil)
		exp := []byte("value1.value2..value4..")
		if !bytes.Equal(text, exp) {
			t.Fatalf("want: %s got: %s", exp, text)
		}
	})
}

func BenchmarkParse(b *testing.B) {
	text := []byte("value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value,value")
	b.Run("bench", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			ll := LogLine{FieldSeparator: 44}
			ll.Parse(text, nil)
		}
	})
}

func BenchmarkToText(b *testing.B) {
	b.Run("bench", func(b *testing.B) {
		ll := LogLine{FieldSeparator: 44}
		for i := 0; i < 100; i++ {
			ll.Set(FieldIndex(i), []byte("value"))
		}
		for n := 0; n < b.N; n++ {
			_ = ll.ToText(nil)
		}
	})
}
