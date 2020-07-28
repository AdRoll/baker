// Package awsutils provides aws-specific types and functions.
package awsutils

var regions = map[string]struct{}{
	"us-east-1":      {},
	"us-west-1":      {},
	"us-west-2":      {},
	"eu-west-1":      {},
	"eu-central-1":   {},
	"ap-southeast-1": {},
	"ap-northeast-1": {},
	"ap-southeast-2": {},
	"ap-northeast-2": {},
	"sa-east-1":      {},
}

// IsValidRegion reports whether a region is a valid aws region identifier.
func IsValidRegion(region string) bool {
	_, ok := regions[region]
	return ok
}
