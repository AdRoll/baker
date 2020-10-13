// Package output provides output components
package output

import (
	"github.com/AdRoll/baker"
)

// All is the list of all baker outputs.
var All = []baker.OutputDesc{
	DynamoDBDesc,
	FileWriterDesc,
	NopDesc,
	OpLogDesc,
	StatsDesc,
	WebSocketDesc,
}
