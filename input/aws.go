package input

import (
	"time"

	"github.com/jpillora/backoff"
)

// TODO[opensource] Move to a AWS-specific package, together with region-related constants
var awsDefaultBackoff = backoff.Backoff{
	Min:    1 * time.Second,
	Max:    10 * time.Second,
	Factor: 2,
	Jitter: true,
}
