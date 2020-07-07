package output

import (
	"github.com/AdRoll/baker"
)

// AllOutputs returns the list of all supported outputs.
func AllOutputs() []baker.OutputDesc {
	return []baker.OutputDesc{
		DynamoDBDesc,
		FilesDesc,
		NopDesc,
		OpLogDesc,
		WebSocketDesc,
	}
}
