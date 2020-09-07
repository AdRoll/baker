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
			name:    "all defaults",
			cfg:     &FileWriterConfig{},
			wantErr: false,
		},
		{
			name: "with split / no fields",
			cfg: &FileWriterConfig{
				PathString: "/path/{{.Field0}}/file.gz",
			},
			wantErr: true,
		},
		{
			name: "with split / with fields",
			cfg: &FileWriterConfig{
				PathString: "/path/{{.Field0}}/file.gz",
			},
			fields:  []baker.FieldIndex{1},
			wantErr: false,
		},
		{
			name: "with region / no config",
			cfg: &FileWriterConfig{
				PathString: "/path/{{.Region}}/file.gz",
			},
			wantErr: true,
		},
		{
			name: "with region / with config",
			cfg: &FileWriterConfig{
				PathString: "/path/{{.Region}}/file.gz",
				Region:     "someRegion",
			},
			wantErr: false,
		},
		{
			name: "with instanceId / no config",
			cfg: &FileWriterConfig{
				PathString: "/path/{{.Instance}}/file.gz",
			},
			wantErr: true,
		},
		{
			name: "with instanceId / with config",
			cfg: &FileWriterConfig{
				PathString: "/path/{{.Instance}}/file.gz",
				InstanceID: "id1",
			},
			wantErr: false,
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
