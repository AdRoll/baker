package input

import (
	"testing"

	"github.com/AdRoll/baker"
)

func TestSQSParseMessage(t *testing.T) {
	tests := []struct {
		format     string
		message    string
		expression string
		wantPath   string
		wantErr    bool
	}{
		{
			format:   "plain",
			message:  "s3://some-bucket/with/stuff/inside",
			wantPath: "s3://some-bucket/with/stuff/inside",
		},
		{
			format: "sns",
			message: `{
				"Type" : "Notification",
				"Message" : "s3://another-bucket/path/to/file",
				"Timestamp" : "2023-05-22T23:21:09.550Z"
			}`,
			wantPath: "s3://another-bucket/path/to/file",
		},
		{
			format:     "json",
			expression: "Foo.Bar",
			message: `{
				"Type": "Notification",
				"Foo": {
				  "Bar": "s3://another-bucket/path/to/file"
				}
			  }`,
			wantPath: "s3://another-bucket/path/to/file",
		},
	}
	for _, tt := range tests {
		t.Run(string(tt.format), func(t *testing.T) {
			in, err := NewSQS(baker.InputParams{
				ComponentParams: baker.ComponentParams{
					DecodedConfig: &SQSConfig{
						MessageFormat:     string(tt.format),
						MessageExpression: tt.expression,
					},
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			s := in.(*SQS)
			path, err := s.parse(tt.message)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseMessage() error = %q, wantErr %t", err, tt.wantErr)
			}
			if path != tt.wantPath {
				t.Errorf("parseMessage() path = %q, want %q", path, tt.wantPath)
			}
		})
	}
}
