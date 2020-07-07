package inpututils

import (
	"net/url"
	"reflect"
	"testing"
)

const DefaultS3Scheme = "s3"

func TestS3ListChoosePathComponents(t *testing.T) {
	var testCases = []struct {
		// Input test path
		path string
		// Expected values
		scheme       string
		bucket       string
		key          string
		isSuccessful bool
	}{
		{"s3://bucket1/dir1/file1", DefaultS3Scheme, "bucket1", "dir1/file1", true},
		{"s3://bucket2/dir2/subdir2/file2", DefaultS3Scheme, "bucket2", "dir2/subdir2/file2", true},
		{"s3://bucket3/dir3/subdir3/subsubdir3/file3", DefaultS3Scheme, "bucket3", "dir3/subdir3/subsubdir3/file3", true},
		{"s3://bucket5/dir /  5", DefaultS3Scheme, "bucket5", "dir /  5", true},
		{"s3a://bucket5/dir /  5", DefaultS3Scheme + "a", "bucket5", "dir /  5", true},
		{"s3n://bucket5/dir /  5", DefaultS3Scheme + "n", "bucket5", "dir /  5", true},
		{"s3://bucket4", "", "", "", false},
		{"s3://buck e t     5/dir /  5", "", "", "", false},
		{"s:3://bucket6/dir/6", "", "", "", false},
		{"s://:3://bucket7/dir/7", "", "", "", false},
		{"dadwdwdwq  :dqw//Dwqdwqdwq   	", "", "", "", false},
	}
	s3Input := NewS3Input("some-region", "some-bucket")
	for _, testCase := range testCases {
		path := testCase.path
		expectedBucket := testCase.bucket
		expectedKey := testCase.key
		expectedScheme := testCase.scheme
		expectedIsSuccessful := testCase.isSuccessful
		t.Run(path, func(t *testing.T) {
			scheme, bucket, key, err := s3Input.choosePathComponents(path)
			isSuccessful := true
			if err != nil {
				isSuccessful = false
			}
			if expectedBucket != bucket || expectedKey != key || expectedIsSuccessful != isSuccessful || expectedScheme != scheme {
				t.Errorf("For path '%s' scheme '%s' bucket '%s' key '%s' success '%t' were expected, got '%s', '%s', '%s','%t'", path, expectedScheme, expectedBucket, expectedKey, expectedIsSuccessful, scheme, bucket, key, isSuccessful)
			}
		})
	}
}

func TestS3ListChoosePathComponentsErrType(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want error
	}{
		{"wrong url ex.1", "3://some-bucket", &url.Error{}},
		{"wrong url ex.2", "3:/some-bucket", &url.Error{}},
		{"wrong url ex.3", "s3 :/   /s  ome-bucket", &url.Error{}},
		{"unsupported url ex.1", "s3://some-bucket", errUnsupportedURLScheme("")},
		{"unsupported url ex.2", "s:o:m://e-bucket", errUnsupportedURLScheme("")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s3Input := NewS3Input("some-region", "some-bucket")
			_, _, _, err := s3Input.choosePathComponents(tt.url)
			if err == nil {
				t.Errorf("url: %s, want: error, got: nil", tt.url)
			}
			if reflect.TypeOf(err) != reflect.TypeOf(tt.want) {
				t.Errorf("url: %s, want: %T, got: %T", tt.url, tt.want, err)
			}
		})
	}
}
