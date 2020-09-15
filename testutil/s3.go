package testutil

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/awstesting/unit"
	"github.com/aws/aws-sdk-go/service/s3"
)

type MockS3 struct {
	DataFn func(interface{})
}

// MockS3Service returns a mocked s3.S3 service which records all operations
// related to Upload S3 API calls.
//
// Once all interactions with the returned service have ended, and not before
// that, ops and params can be accessed. ops and params will hold the list of
// AWS S3 API calls and their parameters. For instance, if ops[0] is "PutObject"
// then params[0] is a *s3.PutObjectInput.
func MockS3Service(wantErr bool, mock *MockS3) (svc *s3.S3, ops *[]string, params *[]interface{}) {
	const respMsg = `<?xml version="1.0" encoding="UTF-8"?>
	<CompleteMultipartUploadOutput>
	   <Location>mockValue</Location>
	   <Bucket>mockValue</Bucket>
	   <Key>mockValue</Key>
	   <ETag>mockValue</ETag>
	</CompleteMultipartUploadOutput>`

	var m sync.Mutex

	ops = &[]string{}
	params = &[]interface{}{}

	partNum := 0
	svc = s3.New(unit.Session)
	svc.Handlers.Unmarshal.Clear()
	svc.Handlers.UnmarshalMeta.Clear()
	svc.Handlers.UnmarshalError.Clear()
	svc.Handlers.Send.Clear()
	svc.Handlers.Send.PushBack(func(r *request.Request) {
		m.Lock()
		defer m.Unlock()

		*ops = append(*ops, r.Operation.Name)
		*params = append(*params, r.Params)

		if wantErr {
			r.HTTPResponse = &http.Response{
				StatusCode: 400,
			}
			return
		}

		r.HTTPResponse = &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte(respMsg))),
		}

		if mock == nil {
			mock = &MockS3{
				DataFn: func(d interface{}) {
					switch data := d.(type) {
					case *s3.CreateMultipartUploadOutput:
						data.UploadId = aws.String("UPLOAD-ID")
					case *s3.UploadPartOutput:
						partNum++
						data.ETag = aws.String(fmt.Sprintf("ETAG%d", partNum))
					case *s3.CompleteMultipartUploadOutput:
						data.Location = aws.String("https://location")
						data.VersionId = aws.String("VERSION-ID")
					case *s3.PutObjectOutput:
						data.VersionId = aws.String("VERSION-ID")
					}
				},
			}
		}

		mock.DataFn(r.Data)
	})

	return svc, ops, params
}
