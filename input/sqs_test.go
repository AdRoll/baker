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
			message: `
			{
				"Type" : "Notification",
				"Message" : "s3://another-bucket/path/to/file",
				"Timestamp" : "2023-05-22T23:21:09.550Z"
			}`,
			wantPath: "s3://another-bucket/path/to/file",
		},
		{
			format:     "json",
			expression: "Foo.Bar",
			message: `
			{
				"Type": "Notification",
				"Foo": {
				  "Bar": "s3://another-bucket/path/to/file"
				}
			}`,
			wantPath: "s3://another-bucket/path/to/file",
		},
		{
			format:     "s3::ObjectCreated",
			expression: "Records[*].join('/',['s3:/', s3.bucket.name, s3.object.key]) | [0]",
			message: `
			{
				"Records": [
					{
						"eventVersion": "2.1",
						"eventSource": "aws:s3",
						"awsRegion": "us-west-2",
						"eventTime": "2021-08-29T11:52:17.371Z",
						"eventName": "ObjectCreated:Put",
						"userIdentity": {
							"principalId": "REDACTED"
						},
						"requestParameters": {
							"sourceIPAddress": "172.18.206.6"
						},
						"responseElements": {
							"x-amz-request-id": "REDACTED",
							"x-amz-id-2": "REDACTEDREDACTEDREDACTEDREDACTEDREDACTED"
						},
						"s3": {
							"s3SchemaVersion": "1.0",
							"configurationId": "tf-s3-topic-20210825REDACTED",
							"bucket": {
								"name": "mybucket",
								"ownerIdentity": {
									"principalId": "A3SX25GZ0Y2AT2"
								},
								"arn": "arn:aws:s3:::mybucket"
							},
							"object": {
								"key": "path/to/a/csv/file/in/a/bucket/file.csv.log.zst",
								"size": 88190,
								"eTag": "9103b07ce4308641b8b7dd6491155eae",
								"sequencer": "00612B74F551DAD52A"
							}
						}
					}
				]
			}`,
			wantPath: "s3://mybucket/path/to/a/csv/file/in/a/bucket/file.csv.log.zst",
		},
		{
			format:     "json",
			expression: "Records[*].join('/',['s3:/', s3.bucket.name, s3.object.key]) | [0]",
			message: `
			{
				"Records": [
					{
						"eventVersion": "2.1",
						"eventSource": "aws:s3",
						"awsRegion": "us-west-2",
						"eventTime": "2021-08-29T11:52:17.371Z",
						"eventName": "ObjectCreated:Put",
						"userIdentity": {
							"principalId": "REDACTED"
						},
						"requestParameters": {
							"sourceIPAddress": "172.18.206.6"
						},
						"responseElements": {
							"x-amz-request-id": "REDACTED",
							"x-amz-id-2": "REDACTEDREDACTEDREDACTEDREDACTEDREDACTED"
						},
						"s3": {
							"s3SchemaVersion": "1.0",
							"configurationId": "tf-s3-topic-20210825REDACTED",
							"bucket": {
								"name": "mybucket",
								"ownerIdentity": {
									"principalId": "A3SX25GZ0Y2AT2"
								},
								"arn": "arn:aws:s3:::mybucket"
							},
							"object": {
								"key": "path/to/a/csv/file/in/a/bucket/file.csv.log.zst",
								"size": 88190,
								"eTag": "9103b07ce4308641b8b7dd6491155eae",
								"sequencer": "00612B74F551DAD52A"
							}
						}
					}
				]
			}`,
			wantPath: "s3://mybucket/path/to/a/csv/file/in/a/bucket/file.csv.log.zst",
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
