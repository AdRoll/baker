package filter

import (
	"bytes"
	"testing"

	"github.com/AdRoll/baker"
)

func TestClearFieldsFilter(t *testing.T) {
	fieldByName := func(name string) (baker.FieldIndex, bool) {
		switch name {
		case "age":
			return 0, true
		case "name":
			return 1, true
		case "surname":
			return 2, true
		}
		return 0, false
	}

	params := baker.FilterParams{
		ComponentParams: baker.ComponentParams{
			FieldByName: fieldByName,
			DecodedConfig: &ClearFieldsConfig{
				Fields: []string{"name", "age", "surname"},
			},
		},
	}

	ll := baker.LogLine{FieldSeparator: ','}
	err := ll.Parse([]byte("12,Jim,Morrison,Paris,"), nil)
	if err != nil {
		t.Fatal(err)
	}

	f, err := NewClearFields(params)
	if err != nil {
		t.Fatal(err)
	}

	f.Process(&ll, func(baker.Record) {})

	for i := baker.FieldIndex(0); i < 3; i++ {
		if ll.Get(i) != nil {
			t.Errorf("field %d not cleared, got %q, want nil", i, ll.Get(i))
		}
	}

	if !bytes.Equal(ll.Get(3), []byte("Paris")) {
		t.Errorf("field 3 = %q, want %q", ll.Get(3), "Paris")
	}
}
