package awsutils

import (
	"time"

	"github.com/jpillora/backoff"
)

// DefaultBackoff is an exponential backoff counter with jitter enabled.
var DefaultBackoff = backoff.Backoff{
	Min:    1 * time.Second,
	Max:    10 * time.Second,
	Factor: 2,
	Jitter: true,
}
