package filter

import (
	"net/url"
	"testing"

	"github.com/AdRoll/baker"
	"github.com/AdRoll/baker/input/inpututils"
)

func TestMetadataUrl(t *testing.T) {
	URL := func(s string) *url.URL {
		u, err := url.Parse(s)
		if err != nil {
			t.Fatalf("cannot parse url: %q", s)
		}
		return u
	}
	tests := []struct {
		name      string
		url       *url.URL
		urlNotSet bool // skip url setting
		dst       string
		want      string
		wantErr   bool
	}{
		{
			name: "url set",
			url:  URL("https://example.com/foo/bar"),
			dst:  "f2",
			want: "https://example.com/foo/bar",
		},
		{
			name:      "url not set",
			urlNotSet: true,
			dst:       "f2",
			want:      "",
		},
		{
			name: "url pointer nil",
			url:  nil,
			dst:  "f2",
			want: "",
		},

		// errors
		{
			name:    "field not exist",
			dst:     "not-exist",
			wantErr: true,
		},
	}

	fieldByName := func(name string) (baker.FieldIndex, bool) {
		switch name {
		case "f1":
			return 0, true
		case "f2":
			return 1, true
		case "f3":
			return 2, true
		}
		return 0, false
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := NewMetadataUrl(baker.FilterParams{
				ComponentParams: baker.ComponentParams{
					FieldByName: fieldByName,
					DecodedConfig: &MetadataUrlConfig{
						DstField: tt.dst,
					},
				},
			})

			if (err != nil) != (tt.wantErr) {
				t.Fatalf("got error = %v, want error = %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			rec1 := &baker.LogLine{FieldSeparator: ','}
			var bakerMetadata baker.Metadata
			if !tt.urlNotSet {
				bakerMetadata = baker.Metadata{inpututils.MetadataURL: tt.url}
			}
			if err := rec1.Parse([]byte{}, bakerMetadata); err != nil {
				t.Fatalf("parse error: %q", err)
			}

			f.Process(rec1, func(rec2 baker.Record) {
				id, ok := fieldByName(tt.dst)
				if !ok {
					t.Fatalf("cannot find field name")
				}
				urlStr := string(rec2.Get(id))
				if urlStr != tt.want {
					t.Errorf("got UnixTime %q, want %q", urlStr, tt.want)
				}
			})
		})
	}
}

func TestMetadataUrlCopy(t *testing.T) {
	url, err := url.Parse("https://example.com/foo/bar")
	if err != nil {
		t.Fatalf("cannot parse url")
	}

	f, err := NewMetadataUrl(baker.FilterParams{
		ComponentParams: baker.ComponentParams{
			FieldByName: func(name string) (baker.FieldIndex, bool) {
				if name == "f1" {
					return 0, true
				}
				return 0, false
			},
			DecodedConfig: &MetadataUrlConfig{
				DstField: "f1",
			},
		},
	})
	if err != nil {
		t.Fatalf("got error = %v, want nil", err)
	}

	bakerMetadata := baker.Metadata{inpututils.MetadataURL: url}
	rec1 := &baker.LogLine{FieldSeparator: ','}
	if err := rec1.Parse([]byte{}, bakerMetadata); err != nil {
		t.Fatalf("parse error: %q", err)
	}
	rec2 := &baker.LogLine{FieldSeparator: ','}
	if err := rec2.Parse([]byte{}, bakerMetadata); err != nil {
		t.Fatalf("parse error: %q", err)
	}

	f.Process(rec1, func(r baker.Record) {
		b := r.Get(0)
		b[0] = 'x'
	})
	f.Process(rec2, func(r baker.Record) {
		if r.Get(0)[0] == 'x' {
			t.Errorf("record does not own bytes after MetadataUrl filter")
		}
	})
}

var sink interface{}

var benchmarkUrlParseData = []struct {
	name string
	url  string
}{
	{

		name: "example",
		url:  "http://www.example.com/bell/breath/back/arithmetic.aspx",
	},
	{
		name: "example-long",
		url:  "https://www.example.com/bell/breath/back/bell/breath/bell/breath/bell/breath/bell/breath/bell/breath/bell/breath/bell/breath/bell/breath/bell/breath/bell/breath/bell/breath/arithmetic.aspx?arithmetic=bite&bit=basin&bit1=basin&bit2=basin&bit3=basin&bit4=basin&bit=basin345&bit8=basin&bit10=basin#baseballallallall",
	},
	{
		name: "long-google",
		url:  "http://chart.apis.google.com/chart?chs=500x500&chma=0,0,100,100&cht=p&chco=FF0000%2CFFFF00%7CFF8000%2C00FF00%7C00FF00%2C0000FF&chd=t%3A122%2C42%2C17%2C10%2C8%2C7%2C7%2C7%2C7%2C6%2C6%2C6%2C6%2C5%2C5&chl=122%7C42%7C17%7C10%7C8%7C7%7C7%7C7%7C7%7C6%7C6%7C6%7C6%7C5%7C5&chdl=android%7Cjava%7Cstack-trace%7Cbroadcastreceiver%7Candroid-ndk%7Cuser-agent%7Candroid-webview%7Cwebview%7Cbackground%7Cmultithreading%7Candroid-source%7Csms%7Cadb%7Csollections%7Cactivity|Chart",
	},
	{
		name: "data-url",
		url:  "data:image/png;base64,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",
	},
}

func BenchmarkUrlParse(b *testing.B) {
	for _, bb := range benchmarkUrlParseData {
		b.Run(bb.name, func(b *testing.B) {
			b.ReportAllocs()

			u, e := url.Parse(bb.url)
			if e != nil {
				b.Fatalf("Url parse error: %v", e)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				sink = u.String()
			}
		})
	}
}

func BenchmarkUrlCopy(b *testing.B) {
	for _, bb := range benchmarkUrlParseData {
		b.Run(bb.name, func(b *testing.B) {
			b.ReportAllocs()

			_, e := url.Parse(bb.url)
			if e != nil {
				b.Fatalf("Url parse error: %v", e)
			}
			u := []byte(bb.url)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				sink := make([]byte, len(u))
				copy(sink, u)
			}
		})
	}
}
