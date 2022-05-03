package baker

import (
	"fmt"
	"testing"

	"github.com/rasky/toml"
)

func TestSizeBytes(t *testing.T) {
	tests := []struct {
		name       string
		tomlString string
		want       SizeBytes // want value in bytes
		wantErr    bool
	}{
		{
			name:       "positive int",
			tomlString: `123456`,
			want:       123456,
		},
		{
			name:       "positive float",
			tomlString: `123.456`,
			want:       123,
		},
		{
			name:       "string no unit",
			tomlString: `"123456"`,
			want:       123456,
		},
		{
			name:       "string IEC unit",
			tomlString: `"123GiB"`,
			want:       123 * 1024 * 1024 * 1024,
		},
		{
			name:       "string SI unit",
			tomlString: `"123GB"`,
			want:       123 * 1000 * 1000 * 1000,
		},

		// default value
		{
			name:       "zero int",
			tomlString: `0`,
			want:       0,
		},
		{
			name:       "zero float",
			tomlString: `0.0`,
			want:       0,
		},
		{
			name:       "zero string",
			tomlString: `"0"`,
			want:       0,
		},
		{
			name:       "empty string",
			tomlString: `""`,
			want:       0,
		},

		// errors
		{
			name:       "negative int",
			tomlString: `-123456`,
			wantErr:    true,
		},
		{
			name:       "negative float",
			tomlString: `-123.456`,
			wantErr:    true,
		},
		{
			name:       "overflow float",
			tomlString: `100000000000000000000`, // 100000000000000000000
			wantErr:    true,
		},
		{
			name:       "unparsable string",
			tomlString: `"some string"`,
			wantErr:    true,
		},
		{
			name:       "partially parsable string",
			tomlString: `"123 some string"`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val := struct{ Field SizeBytes }{}
			_, err := toml.Decode(fmt.Sprintf("\nfield = %s", tt.tomlString), &val)
			if (err != nil) != tt.wantErr {
				t.Fatalf("got: %v, wantErr: %t", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}

			if got := val.Field; got != SizeBytes(tt.want) {
				t.Errorf("Field = SizeBytes(%v), want SizeBytes(%v)", got, tt.want)
			}
		})
	}
}
