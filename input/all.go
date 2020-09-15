package input

import (
	"github.com/AdRoll/baker"
)

// All is the list of all baker inputs.
var All = []baker.InputDesc{
	KCLDesc,
	KTailDesc,
	ListDesc,
	SQSDesc,
	TCPDesc,
}
