package filter

import (
	"bytes"
	"errors"
	"testing"

	"github.com/AdRoll/baker"
)

func TestURLParam(t *testing.T) {
	tests := []struct {
		name          string
		srcField      string
		dstField      string
		param         string
		record        string
		want          string
		wantErr       interface{}
		wantConfigErr bool
	}{
		{
			name:     "default",
			record:   "https://app.adroll.com/?parameter_a=value_a,s1",
			want:     "https://app.adroll.com/?parameter_a=value_a,value_a",
			param:    "parameter_a",
			srcField: "field1",
			dstField: "field2",
		},
		{
			name:     "same destination",
			record:   "https://app.adroll.com/?parameter_a=value_a,s1",
			want:     "value_a,s1",
			param:    "parameter_a",
			srcField: "field1",
			dstField: "field1",
		},
		{
			name:     "partial url",
			record:   "/home?parameter_a=value_a,s1",
			want:     "value_a,s1",
			param:    "parameter_a",
			srcField: "field1",
			dstField: "field1",
		},
		{
			name:     "srcField not a valid url",
			record:   " http://foo.com,s1",
			wantErr:  &ErrURLParamInvalidURL{},
			want:     " http://foo.com,s1", // unchanged
			param:    "parameter_a",
			srcField: "field1",
			dstField: "field1",
		},
		{
			name:     "url param not found",
			record:   "https://app.adroll.com/?parameter_a=value_a,s1",
			want:     "https://app.adroll.com/?parameter_a=value_a,s1", // unchanged
			wantErr:  new(ErrURLParamNotFound),
			param:    "not_parameter_a",
			srcField: "field1",
			dstField: "field1",
		},

		// Configuration errors
		{
			name:          "unknown srcField",
			record:        "https://app.adroll.com/?parameter_a=value_a,s1",
			want:          "",
			param:         "parameter_a",
			srcField:      "s10",
			dstField:      "field1",
			wantConfigErr: true,
		},
		{
			name:          "unknown dstField",
			record:        "https://app.adroll.com/?parameter_a=value_a,s1",
			want:          "",
			param:         "parameter_a",
			srcField:      "field1",
			dstField:      "s10",
			wantConfigErr: true,
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
			if tt.wantConfigErr {
				if err == nil {
					t.Fatal("Expected conf err")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}

			err = f.Process(&ll)
			if (err != nil) != (tt.wantErr != nil) {
				t.Fatalf("Process returned err='%v' wantErr = %t", err, tt.wantErr != nil)
			}
			if tt.wantErr != nil && !errors.As(err, tt.wantErr) {
				t.Fatalf("Process returned err='%v' wantErr='%v'", err, tt.wantErr)
			}

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
