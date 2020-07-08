package input

import (
	"fmt"
	"testing"

	"github.com/AdRoll/baker/testutil"
)

func TestParseMessagePlain(t *testing.T) {
	testutil.InitLogger()
	S3Bucket := ""
	Cfg := &SQSConfig{
		MessageFormat: "plain",
		Bucket:        S3Bucket,
	}

	s := &SQS{
		Cfg: Cfg,
	}

	Message := "s3://some-bucket/log/2015-01-23/l-20150123.gz"
	ActualPath, ActualTs, err := s.parseMessage(&Message, "")
	assertEqual(t, Message, ActualPath)
	assertEqual(t, "", ActualTs)
	assertEqual(t, nil, err)
}

func TestParseMessageSNSFullS3Url(t *testing.T) {
	testutil.InitLogger()
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
	ActualPath, ActualTs, err := s.parseMessage(&Message, "")
	assertEqual(t, ExpectedPath, ActualPath)
	assertEqual(t, ExpectedTs, ActualTs)
	assertEqual(t, nil, err)
}

func TestParseMessageSNSParsedUrl(t *testing.T) {
	testutil.InitLogger()
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
	ActualPath, ActualTs, err := s.parseMessage(&Message, "")
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
