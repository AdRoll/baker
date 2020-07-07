package outputtest

import "github.com/AdRoll/baker"

var RecorderDesc = baker.OutputDesc{
	Name:   "Recorder",
	New:    NewRecorder,
	Config: &RecorderConfig{},
	Raw:    true,
}

// A RecorderConfig specifies the Recorder configuration.
type RecorderConfig struct{}

// A Recorder output appends all received log lines, useful for examination in tests.
type Recorder struct {
	LogLines []baker.OutputLogLine
}

// NewRecorder returns a new Recorder output.
func NewRecorder(cfg baker.OutputParams) (baker.Output, error) {
	return &Recorder{}, nil
}

// Run implements baker.Output interface.
func (r *Recorder) Run(input <-chan baker.OutputLogLine, _ chan<- string) {
	for lldata := range input {
		r.LogLines = append(r.LogLines, lldata)
	}
}

// Stats implements baker.Output interface.
func (r *Recorder) Stats() baker.OutputStats { return baker.OutputStats{} }

// CanShard implements baker.Output interface.
func (r *Recorder) CanShard() bool { return false }
