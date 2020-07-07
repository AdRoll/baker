package input

import (
	"github.com/AdRoll/baker"
)

// AllInputs returns the list of all supported inputs.
func AllInputs() []baker.InputDesc {
	return []baker.InputDesc{
		KTailDesc,
		ListDesc,
		SQSDesc,
		TCPDesc,
	}
}
