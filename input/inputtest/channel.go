package inputtest

import (
	"github.com/AdRoll/baker"
)

// ChannelDesc describes the Channel input. This input is made for testing.
var ChannelDesc = baker.InputDesc{
	Name: "Channel",
	New: func(cfg baker.InputParams) (baker.Input, error) {
		ch := make(Channel)
		return &ch, nil
	},
	Config: &struct{}{},
}

// A Channel input is a Baker input used for testing.
// It's a channel tests can use to send record blobs (baker.Data)
type Channel chan baker.Data

func (in *Channel) Run(output chan<- *baker.Data) error {
	for data := range *in {
		// Copy baker.Data to avoid a race condition.
		buf := make([]byte, len(data.Bytes))
		copy(buf, data.Bytes)
		output <- &baker.Data{Bytes: buf}
	}
	return nil
}

func (in *Channel) Stop()                           {}
func (in *Channel) FreeMem(data *baker.Data)        {}
func (in *Channel) Stats() (stats baker.InputStats) { return }
