package output

import (
	"github.com/AdRoll/baker"
)

// All is the list of all baker outputs.
var All = []baker.OutputDesc{
	DynamoDBDesc,
	FilesDesc,
	NopDesc,
	OpLogDesc,
	WebSocketDesc,
}
