package baker

import (
	"net/url"
	"reflect"
	"sync"
	"testing"
	"time"
)

type dummyInput struct{}

func (d *dummyInput) Run(output chan<- *Data) error {
	return nil
}
func (d *dummyInput) Stats() InputStats {
	return InputStats{}
}
func (d *dummyInput) Stop()              {}
func (d *dummyInput) FreeMem(data *Data) {}

func TestRunFilterChainMetadata(t *testing.T) {
	// Test the same metadata provided by Input can be accessed inside the filters,
	// a simpler version of t.chain was used since the same LogLine received in the chain
	// is passed down to the filters, so we can check there if the same metadata is available.
	rawLine := LogLine{FieldSeparator: DefaultLogLineFieldSeparator}
	rawLine.Set(0, []byte("test"))
	line := rawLine.ToText(nil)
	lastModified := time.Unix(1234, 5678)
	url := &url.URL{
		Scheme: "fake",
		Host:   "fake",
		Path:   "fake"}

	inch := make(chan *Data)
	defer close(inch)
	chainCalled := false
	topo := &Topology{
		// Populate fields needed by runFilterChain
		inch:  inch,
		Input: &dummyInput{},
		linePool: sync.Pool{
			New: func() interface{} {
				return &LogLine{
					FieldSeparator: DefaultLogLineFieldSeparator,
				}
			},
		},
		// Simpler version
		chain: func(l Record) {
			if v, _ := l.Meta("last_modified"); v != lastModified {
				t.Errorf("missing metadata in logline expected last modified = %s got = %s", lastModified, v)
			}
			if v, _ := l.Meta("url"); v != url {
				t.Errorf("missing metadata in logline; expected url = %#v, got #%v", url, v)
			}
			chainCalled = true
		},
	}
	go func() {
		topo.runFilterChain()
		if !chainCalled {
			t.Error("expected Topology.chain to be called.")
		}
	}()

	inch <- &Data{
		Bytes: line,
		Meta: Metadata{
			"last_modified": lastModified,
			"url":           url,
		},
	}
}

func Test_makeUnivocal(t *testing.T) {
	tests := []struct {
		name string
		s    []string
		want []string
	}{
		{
			name: "empty slice",
			s:    []string{},
			want: []string{},
		},
		{
			name: "already univocal",
			s:    []string{"a", "b", "c"},
			want: []string{"a", "b", "c"},
		},
		{
			name: "duplicate",
			s:    []string{"a", "b", "b"},
			want: []string{"a", "b", "b_2"},
		},
		{
			name: "all similar",
			s:    []string{"a", "a", "a"},
			want: []string{"a", "a_2", "a_3"},
		},
		{
			name: "real world case",
			s:    []string{"string_match", "external_match", "not_null", "not_null", "url_escape", "regex_match", "regex_match", "not_null"},
			want: []string{"string_match", "external_match", "not_null", "not_null_2", "url_escape", "regex_match", "regex_match_2", "not_null_3"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			makeUnivocal(tt.s)
			if !reflect.DeepEqual(tt.s, tt.want) {
				t.Errorf("got %+v, want %+v", tt.s, tt.want)
			}
		})
	}
}
