package input

import (
	"testing"
)

func TestSQSParseMessage(t *testing.T) {
	tests := []struct {
		format           sqsFormatType
		message          string
		wantPath, wantTS string
		wantErr          bool
	}{
		{
			format:   sqsFormatPlain,
			message:  "s3://some-bucket/with/stuff/inside",
			wantPath: "s3://some-bucket/with/stuff/inside",
		},
		{
			format: sqsFormatSNS,
			message: `{
				"Type" : "Notification",
				"Message" : "s3://another-bucket/path/to/file",
				"Timestamp" : "2023-05-22T23:21:09.550Z"
			}`,
			wantPath: "s3://another-bucket/path/to/file",
			wantTS:   "2023-05-22T23:21:09.550Z",
		},
	}
	for _, tt := range tests {
		t.Run(string(tt.format), func(t *testing.T) {
			s := SQS{
				Cfg: &SQSConfig{
					format: tt.format,
				},
			}

			path, ts, err := s.parseMessage(&tt.message, nil)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseMessage() error = %q, wantErr %t", err, tt.wantErr)
			}
			if path != tt.wantPath {
				t.Errorf("parseMessage() path = %q, want %q", path, tt.wantPath)
			}
			if ts != tt.wantTS {
				t.Errorf("parseMessage() timestamp = %q, want %q", ts, tt.wantTS)
			}
		})
	}
}

func TestSQSConfig_fillDefaults(t *testing.T) {
	tests := []struct {
		format  string
		expr    string
		want    sqsFormatType
		wantErr bool
	}{
		{format: "", want: sqsFormatSNS},
		{format: "SnS", want: sqsFormatSNS},
		{format: "sns", want: sqsFormatSNS},
		{format: "plain", want: sqsFormatPlain},
		{format: "PLAIN", want: sqsFormatPlain},
		{format: "json", expr: "some_expression", want: sqsFormatJSON},
		{format: "jSON", expr: "some_expression", want: sqsFormatJSON},
		{format: "jSON", wantErr: true},
		{format: " plain", wantErr: true},
		{format: "foobar", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			cfg := &SQSConfig{
				MessageFormat:     tt.format,
				MessageExpression: tt.expr,
			}
			if err := cfg.fillDefaults(); (err != nil) != tt.wantErr {
				t.Fatalf("SQSConfig.fillDefaults() error = %q, wantErr %t", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if cfg.format != tt.want {
				t.Errorf("SQSConfig.fillDefaults() format = %q, want %q", cfg.format, tt.want)
			}
		})
	}
}
