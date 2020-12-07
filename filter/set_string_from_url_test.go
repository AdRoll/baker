package filter

import (
	"net/url"
	"testing"

	"github.com/AdRoll/baker"
	"github.com/AdRoll/baker/input/inpututils"
)

func TestSetStringFromURL(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		strings     []string
		wantDiscard bool   // whether the record should have been discarded
		wantField   string // expected field value in case the record wasn't discaded
	}{
		{
			name:        "string not found in url",
			url:         "s3://no-region",
			strings:     []string{"us-west-2"},
			wantDiscard: true,
		},
		{
			name:        "string found",
			url:         "s3://foo/bar/path-to-us-west-2/us-east-1",
			strings:     []string{"us-west-2", "us-east-1"},
			wantDiscard: false,
			wantField:   "us-west-2",
		},
		{
			name:        "found, multiple strings",
			url:         "s3://foo/bar/path-to-us-west-1",
			strings:     []string{"us-west-2", "us-west-1", "eu-west-2"},
			wantDiscard: false,
			wantField:   "us-west-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Forge a record having tt.url as metadata
			u, err := url.Parse(tt.url)
			if err != nil {
				t.Fatal(err)
			}

			ll := &baker.LogLine{}
			ll.Parse(nil, baker.Metadata{inpututils.MetadataURL: u})

			// Create and setup a filter with tt.strings
			strings := make([][]byte, 0, len(tt.strings))
			for _, s := range tt.strings {
				strings = append(strings, []byte(s))
			}

			nextCount := 0
			f := &SetStringFromURL{field: 0, strings: strings}
			f.Process(ll, func(baker.Record) { nextCount++ })

			if nextCount == 0 != tt.wantDiscard {
				t.Errorf("record discarded=%t, want %t", nextCount == 0, tt.wantDiscard)
			}

			if !tt.wantDiscard {
				if nextCount > 1 {
					t.Errorf("next called %d times, want a single call", nextCount)
				}

				b := ll.Get(0)
				if string(b) != tt.wantField {
					t.Errorf("got field=%q, want %q", b, tt.wantField)
				}
			}
		})
	}
}
