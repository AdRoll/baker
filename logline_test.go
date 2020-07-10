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
	var comma byte = 44
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("!!ALERT!! got panic in Parse: %v", r)
				}
			}()

			b := bytes.Buffer{}
			for i := 0; i < tt.nseps; i++ {
				b.WriteByte(comma)
			}
			ll := LogLine{}
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
		ll := LogLine{}
		ll.Set(i, []byte("myvalue"))
		if !bytes.Contains(ll.ToText(nil), []byte("myvalue")) {
			t.Fatalf("Field %d: %s not found in ll.ToText()", i, "myvalue")
		}
	}
}

func TestLogLineMeta(t *testing.T) {
	var ll Record
	ll = &LogLine{}
	_, ok := ll.Meta("foo")
	if ok {
		t.Errorf("ll.Meta(%q) = _, %v;  want _, false", "foo", ok)
	}

	ll.Parse(nil, &Metadata{"foo": 23})
	val, ok := ll.Meta("foo")
	if !ok || val != 23 {
		t.Errorf("ll.Meta(%q) = %v, %v;  want 23, true", "foo", val, ok)
	}
}

func TestLogLineCache(t *testing.T) {
	var ll Record
	ll = &LogLine{}

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
	RecordConformanceTest(t, &LogLine{})
}
