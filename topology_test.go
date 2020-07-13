package baker

import (
	"net/url"
	"sync"
	"testing"
	"time"
)

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
