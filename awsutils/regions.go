// Package awsutils provides aws-specific types and functions.
package awsutils

// List of available regions.
// source: https://docs.aws.amazon.com/general/latest/gr/s3.html
var regions = map[string]struct{}{
	"us-east-2":      {},
	"us-east-1":      {},
	"us-west-1":      {},
	"us-west-2":      {},
	"af-south-1":     {},
	"ap-east-1":      {},
	"ap-south-1":     {},
	"ap-northeast-3": {},
	"ap-northeast-2": {},
	"ap-southeast-1": {},
	"ap-southeast-2": {},
	"ap-northeast-1": {},
	"ca-central-1":   {},
	"eu-central-1":   {},
	"eu-west-1":      {},
	"eu-west-2":      {},
	"eu-south-1":     {},
	"eu-west-3":      {},
	"eu-north-1":     {},
	"me-south-1":     {},
	"sa-east-1":      {},
}

// IsValidRegion reports whether a region is a valid aws region identifier.
func IsValidRegion(region string) bool {
	_, ok := regions[region]
	return ok
}
