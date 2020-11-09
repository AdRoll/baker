package baker

import (
	"bytes"
	"testing"
)

func TestGenerateMarkdownHelp(t *testing.T) {
	tests := []struct {
		name    string
		desc    interface{}
		want    string
		wantErr bool
	}{
		{
			name:    "nil",
			desc:    nil,
			wantErr: true,
		},
		{
			name:    "unsupported type",
			desc:    23,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &bytes.Buffer{}
			if err := GenerateMarkdownHelp(w, tt.desc); (err != nil) != tt.wantErr {
				t.Errorf("GenerateMarkdownHelp() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotW := w.String(); gotW != tt.want {
				t.Errorf("GenerateMarkdownHelp() = %v, want %v", gotW, tt.want)
			}
		})
	}
}
