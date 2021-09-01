package input

import (
	"fmt"
	"testing"
)

func TestParseMessagePlain(t *testing.T) {
	S3Bucket := ""
	Cfg := &SQSConfig{
		MessageFormat: "plain",
		Bucket:        S3Bucket,
	}

	s := &SQS{
		Cfg: Cfg,
	}

	Message := "s3://some-bucket/log/2015-01-23/l-20150123.gz"
	ActualPath, ActualTs, err := s.parseMessage(&Message, nil)
	assertEqual(t, Message, ActualPath)
	assertEqual(t, "", ActualTs)
	assertEqual(t, nil, err)
}

func TestParseMessageSNSFullS3Url(t *testing.T) {
	S3Bucket := ""
	Cfg := &SQSConfig{
		MessageFormat: "sns",
		Bucket:        S3Bucket,
	}

	s := &SQS{
		Cfg: Cfg,
	}

	Message := `{
  "Type" : "Notification",
  "Message" : "s3://some-bucket/log/2015-01-23/l-20150123.gz",
  "Timestamp" : "2020-05-22T23:21:09.550Z"
}
`
	ExpectedPath := "s3://some-bucket/log/2015-01-23/l-20150123.gz"
	ExpectedTs := "2020-05-22T23:21:09.550Z"
	ActualPath, ActualTs, err := s.parseMessage(&Message, nil)
	assertEqual(t, ExpectedPath, ActualPath)
	assertEqual(t, ExpectedTs, ActualTs)
	assertEqual(t, nil, err)
}

func TestParseMessageSNSParsedUrl(t *testing.T) {
	S3Bucket := "baker-omfg"
	Cfg := &SQSConfig{
		MessageFormat: "sns",
		Bucket:        S3Bucket,
	}

	s := &SQS{
		Cfg: Cfg,
	}

	Message := `{
  "Type" : "Notification",
  "Message" : "s3://some-bucket/log/2015-01-23/l-20150123.gz",
  "Timestamp" : "2020-05-22T23:21:09.550Z"
}
`
	ExpectedPath := "log/2015-01-23/l-20150123.gz"
	ExpectedTs := "2020-05-22T23:21:09.550Z"
	ActualPath, ActualTs, err := s.parseMessage(&Message, nil)
	assertEqual(t, ExpectedPath, ActualPath)
	assertEqual(t, ExpectedTs, ActualTs)
	assertEqual(t, nil, err)
}

func assertEqual(t *testing.T, a interface{}, b interface{}) {
	if a == b {
		return
	}
	t.Fatal(fmt.Sprintf("Assert: %v != %v", a, b))
}

func TestSQSConfig_fillDefaults(t *testing.T) {
	tests := []struct {
		format  string
		want    sqsFormatType
		wantErr bool
	}{
		{format: "", want: sqsFormatSNS},
		{format: "SnS", want: sqsFormatSNS},
		{format: "sns", want: sqsFormatSNS},
		{format: "plain", want: sqsFormatPlain},
		{format: "PLAIN", want: sqsFormatPlain},
		{format: "s3::objectcreated", want: sqsFormatS3ObjectCreated},
		{format: "s3::ObjectCreated", want: sqsFormatS3ObjectCreated},
		{format: " plain", wantErr: true},
		{format: "foobar", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			cfg := &SQSConfig{
				MessageFormat: tt.format,
			}
			if err := cfg.fillDefaults(); (err != nil) != tt.wantErr {
				t.Fatalf("SQSConfig.fillDefaults() error = %q, wantErr %t", err, tt.wantErr)
			}
			if cfg.format != tt.want {
				t.Errorf("SQSConfig.fillDefaults() format = %q, want %q", cfg.format, tt.want)
			}
		})
	}
}
