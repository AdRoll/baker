package filter

import (
	"bytes"
	"testing"

	"github.com/AdRoll/baker"
)

func TestURLParam(t *testing.T) {
	tests := []struct {
		name     string
		srcField string
		dstField string
		param    string
		record   string
		want     string
		wantErr  bool
	}{
		{
			name:     "default",
			record:   "https://app.adroll.com/?parameter_a=value_a,s1",
			want:     "https://app.adroll.com/?parameter_a=value_a,value_a",
			param:    "parameter_a",
			srcField: "field1",
			dstField: "field2",
			wantErr:  false,
		},
		{
			name:     "same destination",
			record:   "https://app.adroll.com/?parameter_a=value_a,s1",
			want:     "value_a,s1",
			param:    "parameter_a",
			srcField: "field1",
			dstField: "field1",
			wantErr:  false,
		},
		{
			name:     "partial url",
			record:   "/home?parameter_a=value_a,s1",
			want:     "value_a,s1",
			param:    "parameter_a",
			srcField: "field1",
			dstField: "field1",
			wantErr:  false,
		},
		{
			name:     "\"srcField\" not a valid url",
			record:   "s0,s1",
			want:     ",s1",
			param:    "parameter_a",
			srcField: "field1",
			dstField: "field1",
			wantErr:  false,
		},
		{
			name:     "\"param\" not in url",
			record:   "https://app.adroll.com/?parameter_a=value_a,s1",
			want:     ",s1",
			param:    "not_parameter_a",
			srcField: "field1",
			dstField: "field1",
			wantErr:  false,
		},
		{
			name:     "unknown \"srcField\"",
			record:   "https://app.adroll.com/?parameter_a=value_a,s1",
			want:     "",
			param:    "parameter_a",
			srcField: "s10",
			dstField: "field1",
			wantErr:  true,
		},
		{
			name:     "unknown \"dstField\"",
			record:   "https://app.adroll.com/?parameter_a=value_a,s1",
			want:     "",
			param:    "parameter_a",
			srcField: "field1",
			dstField: "s10",
			wantErr:  true,
		},
	}

	fieldByName := func(name string) (baker.FieldIndex, bool) {
		switch name {
		case "field1":
			return 0, true
		case "field2":
			return 1, true
		}
		return 0, false
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := baker.FilterParams{
				ComponentParams: baker.ComponentParams{
					FieldByName: fieldByName,
					DecodedConfig: &URLParamConfig{
						SrcField: tt.srcField,
						DstField: tt.dstField,
						Param:    tt.param,
					},
				},
			}

			ll := baker.LogLine{FieldSeparator: ','}
			if err := ll.Parse([]byte(tt.record), nil); err != nil {
				t.Fatal(err)
			}

			f, err := NewURLParam(params)
			if tt.wantErr {
				if err == nil {
					t.Fatal("Expected conf err")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}

			f.Process(&ll, func(baker.Record) {})

			got := ll.ToText(nil)

			wantLine := baker.LogLine{FieldSeparator: ','}
			if err := wantLine.Parse([]byte(tt.want), nil); err != nil {
				t.Fatalf("Cannot parse wanted record: %v", err)
			}

			want := wantLine.ToText(nil)

			if !bytes.Equal(got, want) {
				t.Errorf("got: %s, want: %s", got, want)
			}
		})
	}
}
