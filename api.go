package baker

type InputStats struct {
	NumProcessedLines int64
	CustomStats       map[string]string
	Metrics           MetricsBag
}

type Data struct {
	Bytes []byte
	Meta  Metadata
}

// Metadata about the input data; each Input will directly populate this
// map as appropriate.  Consumers (filters) will access via Get()
type Metadata map[string]interface{}

func (m *Metadata) get(key string) (val interface{}, ok bool) {
	if *m == nil {
		return nil, false
	}

	val, ok = (*m)[key]
	return
}

type FilterStats struct {
	NumProcessedLines int64
	NumFilteredLines  int64
	Metrics           MetricsBag
}

type OutputStats struct {
	NumProcessedLines int64
	NumErrorLines     int64
	CustomStats       map[string]string
	Metrics           MetricsBag
}

type UploadStats struct {
	NumProcessedFiles int64
	NumErrorFiles     int64
	CustomStats       map[string]string
	Metrics           MetricsBag
}

// Input is an interface representing an object that produces
// (fetches) datas for the filter.
type Input interface {
	// Start fetching data and pushing it into the channel.
	// If this call blocks forever, the topology is permanent and
	// acts like a long-running daemon; if this calls exits after
	// it has finished, the topology is meant to be run as a task
	// to process a fixed-size input, and baker will cleanly shutdown
	// after all inputs have been fully processed.
	Run(output chan<- *Data) error

	// Force the input to stop as clean as possible, at a good boundary.
	// This is usually issued at the user's request of exiting the process.
	// For instance, it might make sense to finish processing the current
	// batch of data or the current file, and then save in stable storage
	// the checkpoint to resume it later.
	Stop()

	// Return stats about the input
	Stats() InputStats

	// This function is called when the filter is finished with
	// the memory received through the input channel. Since the
	// memory was allocated by Input, it is returned to it
	// so that it might be recycled.
	FreeMem(data *Data)
}

// Filter is an interface representing one data filter; a filter is a
// function that processes a LogLine and produces a FilterOutput (that is,
// a custom record to be stored somwhere through a Output)
type Filter interface {
	// Process processes a single logline, and then optionally sends it to
	// next filter in the chain.
	// Process might mutate the logline, adding/modifying/removing fields,
	// and might decide to throw it away, or pass it to next filter in chain
	// by calling the next() function. In some cases, a filter might generate
	// multiple loglines in output, by calling next() multiple times.
	// next() is guaranteed to be non-nil; for the last filter of the chain,
	// it points to a function that wraps up the filtering chain and sends
	// the logline to the output.
	Process(l *LogLine, next func(*LogLine))

	// Return stats about the filter
	Stats() FilterStats
}

// Output is an interface representing an object that is processing
// (storing) the output for a filter.
type Output interface {
	// Run processes the OutputLogLine data coming through a channel.
	// Run must block forever.
	// The output implementer will know whether to use clean or raw input
	// channels, only one is actually used at a time.
	Run(in <-chan OutputLogLine, upch chan<- string)

	// Return stats about the output
	Stats() OutputStats

	// Returns true if this output supports sharding
	CanShard() bool
}

// OutputLogLine is the data structure sent to baker output components.
//
// It represents a log line in two possibile formats:
//   * a list of pre-parsed fields, extracted from the log line (as string).
//     This is useful when the output only cares about specific fields and does
//     not need the full log line.
//   * the whole logline, as processed and possibly modified by baker filters (as []byte).
//
// Fields sent to the output are described in the topology. This was designed
// such as an output can work in different modes, by processing different
// fields under the control of the user. Some fields might be required, and
// this validation should be performed by the Output itself. The topology can
// also declare no fields in which case, the Fields slice will be empty.
//
// Line is non-nil only if the output declares itself as a raw output (see
// OutputDesc.Raw). This is done for performance reasons, as recreating the
// whole log line requires allocations and memory copies, and is not always
// required.
type OutputLogLine struct {
	Fields []string // Fields are the fields sent to a Baker output.
	Line   []byte   // Line is the raw log line from which Fields are extracted.
}

// Upload is an interface representing an object that is uploading
// the output of a topology to a configured location
type Upload interface {
	// Run processes the output result as it comes through the channel.
	// Run must block forever
	// upch will receive filenames that Output wants to see uploaded.
	Run(upch <-chan string)

	// Force the upload to stop as cleanly as possible, which usually means
	// to finish up all the existing downloads.
	Stop()

	// Return stats about the upload process
	Stats() UploadStats
}
