package filter

import (
	"bytes"
	"testing"

	"github.com/AdRoll/baker"
)

func TestCrypt(t *testing.T) {
	tests := []struct {
		name   string
		record string

		algorithm       string
		decrypt         bool
		srcField        string
		dstField        string
		algorithmConfig map[string]string

		wantValue []byte
		wantErr   bool
	}{
		{
			name:      "encrypt",
			record:    "s3cr3t,def,ghi",
			algorithm: "fernet",
			decrypt:   false,
			srcField:  "foo",
			dstField:  "foo",
			algorithmConfig: map[string]string{
				"Key": "mp5jfs_We-bngW4srQCwp7nLljXJDyuVCVw1Q8NEo_U=",
			},
			wantValue: []byte("s3cr3t"),
		},
		{
			name:      "decrypt",
			record:    "gAAAAABgAHxm_pT_mAhSxNRb2LHXHZjrIc3eoYLPJxMYGrkRsXrD39EI6fzvs-iwQpiGGesFJ9TagmlBbbhY4NlARAMAIGz90g==,def,ghi",
			algorithm: "fernet",
			decrypt:   true,
			srcField:  "foo",
			dstField:  "foo",
			algorithmConfig: map[string]string{
				"Key": "mp5jfs_We-bngW4srQCwp7nLljXJDyuVCVw1Q8NEo_U=",
			},
			wantValue: []byte("s3cr3t"),
		},
		{
			name:      "decrypt with TTL",
			record:    "gAAAAABgAHxm_pT_mAhSxNRb2LHXHZjrIc3eoYLPJxMYGrkRsXrD39EI6fzvs-iwQpiGGesFJ9TagmlBbbhY4NlARAMAIGz90g==,def,ghi",
			algorithm: "fernet",
			decrypt:   true,
			srcField:  "foo",
			dstField:  "foo",
			algorithmConfig: map[string]string{
				"Key": "mp5jfs_We-bngW4srQCwp7nLljXJDyuVCVw1Q8NEo_U=",
				"TTL": "56307200000", // 200 years in seconds
			},
			wantValue: []byte("s3cr3t"),
		},
		{
			name:      "decrypt error",
			record:    "asdAAABgAHxm_pT_mAhSxNRb2LHXHZjrIc3eoYLPJxMYGrkRsXrD39EI6fzvs-iwQpiGGesFJ9TagmlBbbhY4NlARAMAIGz90g==,def,ghi",
			algorithm: "fernet",
			decrypt:   true,
			srcField:  "foo",
			dstField:  "foo",
			algorithmConfig: map[string]string{
				"Key": "mp5jfs_We-bngW4srQCwp7nLljXJDyuVCVw1Q8NEo_U=",
			},
			wantValue: []byte("asdAAABgAHxm_pT_mAhSxNRb2LHXHZjrIc3eoYLPJxMYGrkRsXrD39EI6fzvs-iwQpiGGesFJ9TagmlBbbhY4NlARAMAIGz90g=="),
		},

		// config errors
		{
			name:      "algorithm not supported",
			algorithm: "not-supported",
			srcField:  "foo",
			dstField:  "foo",
			wantErr:   true,
		},
		{
			name:      "wrong SrcField",
			algorithm: "fernet",
			srcField:  "not-exist",
			dstField:  "foo",
			wantErr:   true,
		},
		{
			name:      "wrong DstField",
			algorithm: "fernet",
			srcField:  "foo",
			dstField:  "not-exist",
			wantErr:   true,
		},
		{
			name:            "fernet config: no Key",
			algorithm:       "fernet",
			srcField:        "foo",
			dstField:        "foo",
			algorithmConfig: map[string]string{},
			wantErr:         true,
		},
		{
			name:      "fernet config: invalid Key",
			algorithm: "fernet",
			srcField:  "foo",
			dstField:  "foo",
			algorithmConfig: map[string]string{
				"Key": "not-a-valid-key",
			},
			wantErr: true,
		},
		{
			name:      "fernet config: TTL not a number",
			algorithm: "fernet",
			srcField:  "foo",
			dstField:  "foo",
			algorithmConfig: map[string]string{
				"Key": "mp5jfs_We-bngW4srQCwp7nLljXJDyuVCVw1Q8NEo_U=",
				"TTL": "not-a-number",
			},
			wantErr: true,
		},
	}

	fieldByName := func(name string) (baker.FieldIndex, bool) {
		switch name {
		case "foo":
			return 0, true
		case "bar":
			return 1, true
		case "baz":
			return 2, true
		}
		return 0, false
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := NewCrypt(baker.FilterParams{
				ComponentParams: baker.ComponentParams{
					FieldByName: fieldByName,
					DecodedConfig: &CryptConfig{
						Decrypt:         tt.decrypt,
						Algorithm:       tt.algorithm,
						SrcField:        tt.srcField,
						DstField:        tt.dstField,
						AlgorithmConfig: tt.algorithmConfig,
					},
				},
			})

			if (err != nil) != (tt.wantErr) {
				t.Fatalf("got error = %v, want error = %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			l := &baker.LogLine{FieldSeparator: ','}
			if err := l.Parse([]byte(tt.record), nil); err != nil {
				t.Fatalf("parse error: %q", err)
			}

			f.Process(l, func(baker.Record) {})

			idx, _ := fieldByName(tt.dstField)
			value := l.Get(idx)

			// When the filter encrypts, as the encrypted value changes every time the encryption
			// is applied, the test can't check the encrypted value against a fixed value,
			// so we need to decrypt it back and check the original value.
			if !tt.decrypt {
				var err error
				value, err = f.(*Crypt).algorithm.decrypt(value)
				if err != nil {
					t.Fatal(err)
				}
			}

			if !bytes.Equal(value, tt.wantValue) {
				t.Errorf("got %q, want %q", value, tt.wantValue)
			}
		})
	}
}
