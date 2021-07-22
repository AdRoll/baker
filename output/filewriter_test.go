package output

import (
	"testing"

	"github.com/AdRoll/baker"
)

func TestFileWriterConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *FileWriterConfig
		fields  []baker.FieldIndex
		wantErr bool
	}{
		{
			name: "all defaults",
			cfg:  &FileWriterConfig{},
		},
		{
			name: "{{.Field0}} and len(output.fields) == 1",
			cfg: &FileWriterConfig{
				PathString: "/path/{{.Field0}}/file.gz",
			},
			fields: []baker.FieldIndex{0},
		},
		{
			name: "{{.Field0}} and len(output.fields) > 1",
			cfg: &FileWriterConfig{
				PathString: "/path/{{.Field0}}/file.gz",
			},
			fields: []baker.FieldIndex{0, 1},
		},

		// error cases
		{
			name: "{{.Field0}} and len(output.fields) == 0",
			cfg: &FileWriterConfig{
				PathString: "/path/{{.Field0}}/file.gz",
			},
			fields:  []baker.FieldIndex{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := baker.OutputParams{
				ComponentParams: baker.ComponentParams{
					DecodedConfig: tt.cfg,
				},
				Fields: tt.fields,
			}
			_, err := NewFileWriter(cfg)
			if tt.wantErr && err == nil {
				t.Fatalf("wantErr: %v, got: %v", tt.wantErr, err)
			}

			if !tt.wantErr && err != nil {
				t.Fatalf("wantErr: %v, got: %v", tt.wantErr, err)
			}
		})
	}
}
