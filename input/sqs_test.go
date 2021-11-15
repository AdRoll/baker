package input

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sqs"
	log "github.com/sirupsen/logrus"

	"github.com/AdRoll/baker"
	"github.com/AdRoll/baker/output/outputtest"
	"github.com/AdRoll/baker/testutil"
)

func TestSQSParseMessage(t *testing.T) {
	tests := []struct {
		name                        string
		format, message, expression string
		wantPath                    string
		wantConfigErr, wantParseErr bool
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

		// config errors
		{
			name:          "json format but empty expression",
			format:        "json",
			expression:    "",
			message:       "whatever",
			wantPath:      "whatever",
			wantConfigErr: true,
		},
		{
			name:          "json format with incorrect jmespath",
			format:        "json",
			expression:    "incorrect jmespath expression ",
			message:       "whatever",
			wantPath:      "whatever",
			wantConfigErr: true,
		},
		{
			format:        "unknown format",
			message:       "whatever",
			wantPath:      "whatever",
			wantConfigErr: true,
		},

		// parse errors
		{
			name:       "invalid json payload",
			format:     "json",
			expression: "Foo.Bar",
			message: `
			{
				"Type": "Notification",
				"Foo": {
				  "Bar": 
				}
			}`,
			wantParseErr: true,
		},
		{
			name:       "field not found",
			format:     "json",
			expression: "Foo.Bar",
			message: `
			{
				"Type": "Notification",
				"Foo": {}
			}`,
			wantParseErr: true,
		},
		{
			name:       "field of wrong type",
			format:     "json",
			expression: "Foo.Bar",
			message: `
			{
				"Type": "Notification",
				"Foo": {
					"Bar": 123456
				}
			}`,
			wantParseErr: true,
		},
	}
	for _, tt := range tests {
		tname := tt.name
		if tname == "" {
			tname = string(tt.format)
		}

		t.Run(tname, func(t *testing.T) {
			in, err := NewSQS(baker.InputParams{
				ComponentParams: baker.ComponentParams{
					DecodedConfig: &SQSConfig{
						MessageFormat:     string(tt.format),
						MessageExpression: tt.expression,
					},
				},
			})
			if (err != nil) != tt.wantConfigErr {
				t.Fatalf("NewSQS() error = %q, wantConfigErr %t", err, tt.wantConfigErr)
			}
			if tt.wantConfigErr {
				return
			}

			s := in.(*SQS)
			path, err := s.parse(tt.message)
			if (err != nil) != tt.wantParseErr {
				t.Fatalf("parseMessage() error = %q, wantParseErr %t", err, tt.wantParseErr)
			}
			if tt.wantParseErr {
				return
			}
			if path != tt.wantPath {
				t.Errorf("parseMessage() path = %q, want %q", path, tt.wantPath)
			}
		})
	}
}

type sqsIntegrationTestCase struct {
	name          string
	queuePrefixes []string
	messages      map[string][]sqs.Message

	wantRecords []string // records we want, order doesn't matter
}

func TestSQS(t *testing.T) {
	if testing.Verbose() {
		testutil.SetLogLevel(t, log.DebugLevel)
	}

	// Return an sqs.Message (can't use struct literal since aws use *string everywhere)
	sqsMessage := func(body string) sqs.Message { return sqs.Message{Body: &body} }

	tests := []sqsIntegrationTestCase{
		{
			name:          "multiple queues and buckets",
			queuePrefixes: []string{"queue-a", "queue-b", "queue-c"},
			messages: map[string][]sqs.Message{
				"queue-a": {
					sqsMessage("s3://bucket-a/path/to/file/1.zst"),
					sqsMessage("s3://bucket-a/path/to/file/2.zst"),
				},
				"queue-b": {
					sqsMessage("s3://bucket-b/path/to/file/1.zst"),
					sqsMessage("s3://bucket-b/path/to/file/2.zst"),
				},
				"queue-c": {
					sqsMessage("s3://bucket-c/path/to/file/1.zst"),
					sqsMessage("s3://bucket-c/path/to/file/2.zst"),
				},
			},
			wantRecords: []string{
				"bucket-a,path/to,1.zst",
				"bucket-a,path/to,2.zst",
				"bucket-b,path/to,1.zst",
				"bucket-b,path/to,2.zst",
				"bucket-c,path/to,1.zst",
				"bucket-c,path/to,2.zst",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, testIntegrationSQS(tt))
	}
}

func testIntegrationSQS(tc sqsIntegrationTestCase) func(t *testing.T) {
	return func(t *testing.T) {
		toml := `
[fields]
Names=["bucket", "path", "filename"]

[csv]
field_separator=","

[input]
Name="sqs"
[input.config]
Bucket="bucket-a"
MessageFormat="plain"
QueuePrefixes=[%v]
FilePathFilter=".*"

[output]
Name="RawRecorder"
Procs=1
fields=["bucket", "path", "filename"]
`

		/* Configure the pipeline */
		comp := baker.Components{
			Inputs:  []baker.InputDesc{SQSDesc},
			Outputs: []baker.OutputDesc{outputtest.RawRecorderDesc},
		}

		prefixes := []string{}
		for _, pref := range tc.queuePrefixes {
			prefixes = append(prefixes, `"`+pref+`"`)
		}

		r := strings.NewReader(fmt.Sprintf(toml, strings.Join(prefixes, ",")))
		cfg, err := baker.NewConfigFromToml(r, comp)
		if err != nil {
			t.Fatal(err)
		}

		topo, err := baker.NewTopologyFromConfig(cfg)
		if err != nil {
			t.Fatal(err)
		}

		// Replace aws services interfaces with mocks.
		topo.Input.(*SQS).svc = &mockSQSClient{
			queues: tc.messages,
		}
		topo.Input.(*SQS).s3Input.SetS3API(newMockedS3FromFS(os.DirFS("testdata/sqstest")))

		/* Run the pipeline */
		topo.Start()
		time.Sleep(2 * time.Second)
		topo.Stop()
		topo.Wait()
		if err := topo.Error(); err != nil {
			t.Fatalf("topology error: %v", err)
		}

		/* Checks */
		out := topo.Output[0].(*outputtest.Recorder)

		want := make(map[string]struct{})
		for _, r := range tc.wantRecords {
			want[r] = struct{}{}
		}

		got := make(map[string]struct{})
		for _, r := range out.Records {
			got[string(r.Record)] = struct{}{}
		}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("got records:\n%+v\n\nwant:\n%+v", got, want)
		}
	}
}

// mockSQSClient emulates the set of aws SQS API interface methods used for the
// SQS input. The 'queues' field should be filled with a map where the key is
// the SQS queue name and the value is a list of messages that queue returns,
// one by one.
type mockSQSClient struct {
	*sqs.SQS

	mu     sync.Mutex
	queues map[string][]sqs.Message
}

func (c *mockSQSClient) ListQueuesWithContext(ctx aws.Context, input *sqs.ListQueuesInput, options ...request.Option) (*sqs.ListQueuesOutput, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var queueURLs []*string
	for name := range c.queues {
		if strings.HasPrefix(name, *input.QueueNamePrefix) {
			u := "https://sqs.us-west-2.amazonaws.com/123456789012/" + name
			queueURLs = append(queueURLs, &u)
		}
	}

	out := &sqs.ListQueuesOutput{QueueUrls: queueURLs}
	log.WithFields(log.Fields{"sqs": "ListQueuesWithContext", "input": *input, "out": *out}).Debug()
	return out, nil
}

// ReceiveMessageWithContext sends the first message for the requested queue, if
// any, then removes it from the queue.
func (c *mockSQSClient) ReceiveMessageWithContext(ctx aws.Context, input *sqs.ReceiveMessageInput, options ...request.Option) (*sqs.ReceiveMessageOutput, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	time.Sleep(50 * time.Millisecond)

	u, err := url.Parse(*input.QueueUrl)
	if err != nil {
		return nil, err
	}
	ucomps := strings.Split(u.Path, "/")
	queueName := ucomps[len(ucomps)-1]
	msgs, ok := c.queues[queueName]
	if !ok {
		return nil, fmt.Errorf("queue %v not found", queueName)
	}

	out := &sqs.ReceiveMessageOutput{
		Messages: []*sqs.Message{},
	}

	if len(msgs) > 0 {
		// Pops the first message out of the queue.
		var first sqs.Message
		first, msgs = msgs[0], msgs[1:]
		out.Messages = append(out.Messages, &first)
		c.queues[queueName] = msgs
	}

	log.WithFields(log.Fields{"sqs": "ReceiveMessageWithContext", "input": *input, "out": *out}).Debug()
	return out, nil
}

// DeleteMessageWithContext does nothing since messages are removed from the
// queue as soon as they're requested.
func (c *mockSQSClient) DeleteMessageWithContext(ctx aws.Context, input *sqs.DeleteMessageInput, options ...request.Option) (*sqs.DeleteMessageOutput, error) {
	return nil, nil
}

// mockedS3FS emulates the GetObject method of an AWS S3 interface by returning
// the content of files from the provided file system.
type mockedS3FS struct {
	*s3.S3
	fs fs.FS
}

func newMockedS3FromFS(fs fs.FS) *mockedS3FS {
	return &mockedS3FS{fs: fs}
}

func (c *mockedS3FS) GetObject(input *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
	log.WithFields(log.Fields{"s3": "GetObject", "input": *input, "out": nil}).Debug()

	path := fmt.Sprintf("%s/%s", *input.Bucket, *input.Key)
	buf, err := fs.ReadFile(c.fs, path)
	if err != nil {
		return nil, fmt.Errorf("mockedS3FS, error s3.GetObject with path %v: %v", path, err)
	}
	fi, err := fs.Stat(c.fs, path)
	if err != nil {
		return nil, fmt.Errorf("mockedS3FS, can't stat %v: %v", path, err)
	}
	mtime := fi.ModTime()
	length := int64(len(buf))

	out := s3.GetObjectOutput{
		Body:          io.NopCloser(bytes.NewReader(buf)),
		ContentLength: &length,
		LastModified:  &mtime,
	}

	return &out, nil
}
