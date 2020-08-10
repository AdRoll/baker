package outputtest

import "github.com/AdRoll/baker"

// RawRecorderDesc describes the RawRecorder debug output.
var RawRecorderDesc = baker.OutputDesc{
	Name:   "RawRecorder",
	New:    NewRecorder,
	Config: &RecorderConfig{},
	Raw:    true,
}

// RecorderDesc describes the Recorder debug output.
var RecorderDesc = baker.OutputDesc{
	Name:   "Recorder",
	New:    NewRecorder,
	Config: &RecorderConfig{},
	Raw:    false,
}

// A RecorderConfig specifies the Recorder configuration.
type RecorderConfig struct{}

// A Recorder output appends all received records, useful for examination in tests.
type Recorder struct {
	Records []baker.OutputRecord
}

// NewRecorder returns a new Recorder output.
func NewRecorder(cfg baker.OutputParams) (baker.Output, error) {
	return &Recorder{}, nil
}

// Run implements baker.Output interface.
func (r *Recorder) Run(input <-chan baker.OutputRecord, _ chan<- string) error {
	for lldata := range input {
		r.Records = append(r.Records, lldata)
	}

	return nil
}

// Stats implements baker.Output interface.
func (r *Recorder) Stats() baker.OutputStats { return baker.OutputStats{} }

// CanShard implements baker.Output interface.
func (r *Recorder) CanShard() bool { return true }
