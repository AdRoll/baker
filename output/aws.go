package output

import (
	"time"

	"github.com/jpillora/backoff"
)

var awsDefaultBackoff = backoff.Backoff{
	Min:    100 * time.Millisecond,
	Max:    10 * time.Second,
	Factor: 2,
	Jitter: true,
}

var awsRegions = map[string]bool{
	"us-east-1":      true,
	"us-west-1":      true,
	"us-west-2":      true,
	"eu-west-1":      true,
	"eu-central-1":   true,
	"ap-southeast-1": true,
	"ap-northeast-1": true,
	"ap-southeast-2": true,
	"ap-northeast-2": true,
	"sa-east-1":      true,
}
