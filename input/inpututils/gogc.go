package inpututils

import (
	"os"
	"runtime/debug"
)

// SetGCPercentIfNotSet sets the GC target percentage, unless GOGC environment
// variable is set, in which case SetGCPercentIfNotSet doesn't not override it
// and let it as is.
func SetGCPercentIfNotSet(percent int) {
	gogc := os.Getenv("GOGC")
	if gogc != "" {
		return
	}

	debug.SetGCPercent(percent)
}
