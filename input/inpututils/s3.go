package inpututils

import (
	"fmt"
	"io"
	"net/url"
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
)

type S3Input struct {
	*CompressedInput

	Bucket string

	svc s3iface.S3API
}

func NewS3Input(region, bucket string) (*S3Input, error) {
	sess, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	if err != nil {
		return nil, fmt.Errorf("can't create aws session: %v", err)
	}

	svc := s3.New(sess)
	s := &S3Input{
		Bucket: bucket,
		svc:    svc,
	}
	s.CompressedInput = NewCompressedInput(s.openS3File, s.sizeS3File, make(chan bool, 1))
	return s, nil
}

// SetS3API allows to replace the S3API, for tests.
func (s3 *S3Input) SetS3API(s3API s3iface.S3API) {
	s3.svc = s3API
}

// ProcessDirectory enqueues all files matching a specific prefix for
// processing by s3Input. If prefix is actually a s3 url use the bucket
// there instead of the one provided at creation time.
//
// This function makes (multiple) remotes call to acquire the listing of
// all files matching the specified prefix in the bucket, and enqueue
// them for processing,
func (s *S3Input) ProcessDirectory(prefix string) error {
	s3Scheme, s3Bucket, s3Key, err := s.choosePathComponents(prefix)
	if err != nil {
		return err
	}
	isFullPath, err := isValidScheme(s3Scheme)
	if err != nil {
		return err
	}
	return s.svc.ListObjectsPages(&s3.ListObjectsInput{
		Bucket: aws.String(s3Bucket),
		Prefix: aws.String(s3Key),
	}, func(page *s3.ListObjectsOutput, lastPage bool) bool {
		for _, o := range page.Contents {
			var key string
			// If prefix is a full s3 url it means we need to provide
			// openS3File with the full path in order to be able to correctly
			// fetch the size and the contents of the file.
			if isFullPath {
				key = fmt.Sprintf("%s://%s/%s", s3Scheme, s3Bucket, *o.Key)
			} else {
				key = *o.Key
			}
			s.ProcessFile(key)
		}
		return true
	})
}

func (s *S3Input) openS3File(fn string) (io.ReadCloser, int64, time.Time, *url.URL, error) {
	_, s3Bucket, s3Key, err := s.choosePathComponents(fn)
	if err != nil {
		return nil, 0, time.Time{}, nil, err
	}

	resp, err := s.svc.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(s3Bucket),
		Key:    aws.String(s3Key),
	})
	if err != nil {
		return nil, 0, time.Time{}, nil, err
	}

	urlObject, err := url.Parse(fn)
	if err != nil {
		return nil, 0, time.Time{}, nil, err
	}

	return resp.Body, *resp.ContentLength, *resp.LastModified, urlObject, nil
}

func (s *S3Input) sizeS3File(fn string) (int64, error) {
	_, s3Bucket, s3Key, err := s.choosePathComponents(fn)
	if err != nil {
		return 0, err
	}

	resp, err := s.svc.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(s3Bucket),
		Key:    aws.String(s3Key),
	})
	if err != nil {
		return 0, err
	}
	return *resp.ContentLength, nil
}

// Check if scheme is s3 or s3a or s3n
func isValidScheme(scheme string) (bool, error) {
	return regexp.MatchString(`^s3[an]?$`, scheme)
}

// errUnsupportedURLScheme is the error indicating a path has an invalid or
// unsupported URL scheme for the S3Input.
type errUnsupportedURLScheme string

// Error implements the error interface.
func (e errUnsupportedURLScheme) Error() string {
	return fmt.Sprintf("%s unsupported, should be s3[a]://BUCKET/DIR_PATH/FILE_NAME or DIR_PATH/FILE_NAME (no scheme)", string(e))
}

// If the path is a full s3/s3a/s3n url the extract the bucket, key and scheme from
// it and return them, else use the bucket from the instance, path as key and
// default scheme "s3"
func (s *S3Input) choosePathComponents(path string) (scheme, bucket, key string, err error) {
	s3Url, err := url.Parse(path)
	scheme = ""
	bucket = ""
	key = ""
	// If the path is url compliant and its scheme is s3 or none and there is a
	// path (downloading a whole bucket is not supported/safe) we can proceed
	// to assign the bucket and path
	if err != nil {
		return "", "", "", err
	}

	validScheme, err := isValidScheme(s3Url.Scheme)
	if err != nil {
		return "", "", "", err
	}

	if s3Url.Path == "" || (s3Url.Scheme != "" && !validScheme) {
		return "", "", "", errUnsupportedURLScheme(path)
	}

	scheme = ""
	bucket = s.Bucket
	key = path

	if validScheme {
		// The bucket is the "host" portion of the s3 url and the key is
		// the "path"
		scheme = s3Url.Scheme
		bucket = s3Url.Host
		key = s3Url.Path[1:]
	}
	return scheme, bucket, key, nil
}
