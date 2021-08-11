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

func TestLogLineParse(t *testing.T) {
	t.Run("default comma separator", func(t *testing.T) {
		text := []byte("value1,value2,,value4")
		ll := LogLine{FieldSeparator: ','}
		ll.Parse(text, nil)

		got, want := ll.Get(0), []byte("value1")
		if !bytes.Equal(got, want) {
			t.Errorf("got: %v want: %v", got, want)
		}
		got, want = ll.Get(1), []byte("value2")
		if !bytes.Equal(got, want) {
			t.Errorf("got: %v want: %v", got, want)
		}
		got, want = ll.Get(2), []byte("")
		if !bytes.Equal(got, want) {
			t.Errorf("got: %v want: %v", got, want)
		}
		got, want = ll.Get(3), []byte("value4")
		if !bytes.Equal(got, want) {
			t.Errorf("got: %v want: %v", got, want)
		}
	})

	t.Run("custom dot separator", func(t *testing.T) {
		text := []byte("value1.value2..value4")
		ll := LogLine{FieldSeparator: '.'}
		ll.Parse(text, nil)

		got, want := ll.Get(0), []byte("value1")
		if !bytes.Equal(got, want) {
			t.Errorf("got: %v want: %v", got, want)
		}
		got, want = ll.Get(1), []byte("value2")
		if !bytes.Equal(got, want) {
			t.Errorf("got: %v want: %v", got, want)
		}
		got, want = ll.Get(2), []byte("")
		if !bytes.Equal(got, want) {
			t.Errorf("got: %v want: %v", got, want)
		}
		got, want = ll.Get(3), []byte("value4")
		if !bytes.Equal(got, want) {
			t.Errorf("got: %v want: %v", got, want)
		}
	})

	t.Run("nil", func(t *testing.T) {
		ll := LogLine{FieldSeparator: '.'}
		ll.Parse(nil, nil)

		for i := FieldIndex(0); i < LogLineNumFields+NumFieldsBaker; i++ {
			got := ll.Get(i)
			if len(got) != 0 {
				t.Errorf("idx=%d: got: %s want nil", i, got)
			}
		}

		got := ll.ToText(nil)
		if len(got) != 0 {
			t.Errorf("got: %v want: nil", got)
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
			t.Errorf("got: %s want: %s", text, exp)
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
			t.Errorf("got: %s want: %s", text, exp)
		}
	})

	t.Run("empty logline", func(t *testing.T) {
		ll := LogLine{FieldSeparator: ','}
		got := ll.ToText(nil)
		if !bytes.Equal(got, []byte("")) {
			t.Errorf("got: %s want: ''", got)
		}
	})

	t.Run("set", func(t *testing.T) {
		want := []byte(",,value2")

		ll := LogLine{FieldSeparator: ','}
		ll.Set(2, []byte("value2"))
		got := ll.ToText(nil)
		if !bytes.Equal(got, want) {
			t.Errorf("got: %s want: %s", got, want)
		}
	})

	t.Run("parse", func(t *testing.T) {
		want := []byte("value,value,value")

		ll := LogLine{FieldSeparator: ','}
		ll.Parse(want, nil)
		got := ll.ToText(nil)
		if !bytes.Equal(got, want) {
			t.Errorf("got: %s want: %s", got, want)
		}
	})

	t.Run("parse with buffer", func(t *testing.T) {
		want := []byte("value,value,value")

		ll := LogLine{FieldSeparator: ','}
		ll.Parse(want, nil)

		// correct size
		buf := make([]byte, 0, len(want))
		got := ll.ToText(buf)
		if !bytes.Equal(got, want) {
			t.Errorf("got: %s, want: %s", got, want)
		}

		// smaller
		buf = make([]byte, 0, len(want)/2)
		got = ll.ToText(buf)
		if !bytes.Equal(got, want) {
			t.Errorf("got: %s, want: %s", got, want)
		}

		// bigger
		buf = make([]byte, 0, len(want)*2)
		got = ll.ToText(buf)
		if !bytes.Equal(got, want) {
			t.Errorf("got: %s, want: %s", got, want)
		}
	})

	t.Run("set with buffer", func(t *testing.T) {
		want := []byte(",,value2,")

		ll := LogLine{FieldSeparator: ','}
		ll.Set(2, []byte("value2"))
		ll.Set(3, []byte(""))

		// correct size
		buf := make([]byte, 0, len(want))
		got := ll.ToText(buf)
		if !bytes.Equal(got, want) {
			t.Errorf("got: %s want: %s", got, want)
		}

		// smaller
		buf = make([]byte, 0, len(want)/2)
		got = ll.ToText(buf)
		if !bytes.Equal(got, want) {
			t.Errorf("got: %s want: %s", got, want)
		}

		// bigger
		buf = make([]byte, 0, len(want)*2)
		got = ll.ToText(buf)
		if !bytes.Equal(got, want) {
			t.Errorf("got: %s want: %s", got, want)
		}
	})

	t.Run("parse and set", func(t *testing.T) {
		want := []byte("value2,value,value")
		text := []byte("value,value,value")

		ll := LogLine{FieldSeparator: ','}
		ll.Parse(text, nil)
		ll.Set(0, []byte("value2"))
		got := ll.ToText(nil)
		if !bytes.Equal(got, want) {
			t.Errorf("got: %s, want: %s", got, want)
		}
	})

	t.Run("parse and set 2", func(t *testing.T) {
		want := []byte("value,value,value,,,value2")
		text := []byte("value,value,value")

		ll := LogLine{FieldSeparator: ','}
		ll.Parse(text, nil)
		ll.Set(5, []byte("value2"))
		got := ll.ToText(nil)
		if !bytes.Equal(got, want) {
			t.Errorf("got: %s want: %s", got, want)
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
		got := ll.ToText(nil)
		if !bytes.Equal(got, want) {
			t.Errorf("got: %s want: %s", got, want)

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
		got := ll.ToText(nil)
		if !bytes.Equal(got, want) {
			t.Errorf("got: %s want: %s", got, want)

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
		got := ll.ToText(nil)
		if !bytes.Equal(got, want) {
			t.Errorf("got: %s want: %s", got, want)
		}
	})

	t.Run("set custom fields", func(t *testing.T) {
		want := []byte{}

		ll := LogLine{FieldSeparator: ','}
		ll.Set(LogLineNumFields, []byte("custom1"))
		ll.Set(LogLineNumFields+1, []byte("custom2"))
		ll.Set(LogLineNumFields+NumFieldsBaker-1, []byte("customN"))
		got := ll.ToText(nil)
		if !bytes.Equal(got, want) {
			t.Errorf("got: %s want: %s", got, want)
		}
	})

	t.Run("parse + set custom", func(t *testing.T) {
		want := []byte("value,value,value")

		ll := LogLine{FieldSeparator: ','}
		ll.Parse(want, nil)
		ll.Set(LogLineNumFields, []byte("custom1"))
		got := ll.ToText(nil)
		if !bytes.Equal(got, want) {
			t.Errorf("got: %s want: %s", got, want)
		}
	})

	t.Run("parse max num fields + set custom", func(t *testing.T) {
		values := make([]string, 0, LogLineNumFields)
		for i := 0; i < int(LogLineNumFields); i++ {
			values = append(values, "value")
		}
		want := []byte(strings.Join(values, ","))

		ll := LogLine{FieldSeparator: ','}
		ll.Parse(want, nil)
		ll.Set(LogLineNumFields+10, []byte("custom10"))
		got := ll.ToText(nil)
		if !bytes.Equal(got, want) {
			t.Errorf("got: %s want: %s", got, want)
		}
	})
}

func TestLogLineCustomFields(t *testing.T) {
	t.Run("parse error to many fields", func(t *testing.T) {
		values := make([]string, 0, LogLineNumFields)
		for i := 0; i < int(LogLineNumFields)+1; i++ {
			values = append(values, "value")
		}
		want := []byte(strings.Join(values, ","))

		ll := LogLine{FieldSeparator: ','}
		err := ll.Parse(want, nil)
		if err != errLogLineTooManyFields {
			t.Errorf("got: %s want: %s", err, errLogLineTooManyFields)
		}
	})

	t.Run("set/get custom fields", func(t *testing.T) {
		ll := LogLine{FieldSeparator: ','}

		want := []byte("custom1")
		idx := LogLineNumFields
		ll.Set(idx, want)
		got := ll.Get(idx)
		if !bytes.Equal(got, want) {
			t.Errorf("got: %s want: %s", got, want)

		}

		want = []byte("customN")
		idx = LogLineNumFields + NumFieldsBaker - 1
		ll.Set(idx, want)
		got = ll.Get(idx)
		if !bytes.Equal(got, want) {
			t.Errorf("got: %s want: %s", got, want)
		}

		// Custom fields should not be serialized.
		got = ll.ToText(nil)
		if !bytes.Equal(got, []byte{}) {
			t.Errorf("got: %s want: ''", got)
		}
	})
}

func TestLogLineGet(t *testing.T) {
	t.Run("zero value + get", func(t *testing.T) {
		ll := LogLine{FieldSeparator: ','}

		for i := FieldIndex(0); i < LogLineNumFields+NumFieldsBaker; i++ {
			got := ll.Get(i)
			if got != nil {
				t.Errorf("ll.Get(%d) = %q, want nil", i, got)
			}
		}
	})

	t.Run("parse + get", func(t *testing.T) {
		ll := LogLine{FieldSeparator: ','}
		ll.Parse([]byte("value,value,value"), nil)

		for i := FieldIndex(3); i < LogLineNumFields+NumFieldsBaker; i++ {
			got := ll.Get(i)
			if len(got) != 0 {
				t.Errorf("ll.Get(%d) = %q, want nil", i, got)
			}
		}
	})

	t.Run("3 x set + get", func(t *testing.T) {
		ll := LogLine{FieldSeparator: ','}
		ll.Set(FieldIndex(0), []byte("value"))
		ll.Set(FieldIndex(1), []byte("value"))
		ll.Set(FieldIndex(2), []byte("value"))

		for i := FieldIndex(3); i < LogLineNumFields+NumFieldsBaker; i++ {
			got := ll.Get(i)
			if len(got) != 0 {
				t.Errorf("ll.Get(%d) = %q, want nil", i, got)
			}
		}
	})

	t.Run("out of range", func(t *testing.T) {
		fidx := LogLineNumFields + NumFieldsBaker

		defer func() {
			if r := recover(); r == nil {
				t.Errorf("Get(%d) should panic", fidx)
			}
		}()

		ll := LogLine{FieldSeparator: ','}
		ll.Get(fidx)
	})
}

func TestLogLineSetPanic(t *testing.T) {
	t.Run("index out of range", func(t *testing.T) {
		fidx := LogLineNumFields + NumFieldsBaker

		defer func() {
			if r := recover(); r == nil {
				t.Errorf("Set(%d) should panic", fidx)
			}
		}()

		ll := LogLine{FieldSeparator: ','}
		ll.Set(fidx, []byte("value"))
	})

	t.Run("max fields written", func(t *testing.T) {
		var fidx FieldIndex

		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Set(%d) should not panic: %q", fidx, r)
			}
		}()

		ll := LogLine{FieldSeparator: ','}
		// Maximum of 254 changed fields.
		for i := 0; i < 255; i++ {
			fidx = FieldIndex(i)
			ll.Set(fidx, []byte("value"))
		}
	})

	t.Run("too many field written", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("Set(%d) should panic", 1000)
			}
		}()

		ll := LogLine{FieldSeparator: ','}
		// Maximum of 254 changed fields.
		for i := 0; i < 255; i++ {
			ll.Set(FieldIndex(i), []byte("value"))
		}
		// One more.
		ll.Set(FieldIndex(1000), []byte("value"))
	})
}
