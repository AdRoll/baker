package baker

// Data represents raw data consumed by a baker input, possibly
// containing multiple records before they're parsed.
type Data struct {
	Bytes []byte   // Bytes is the slice of raw bytes read by an input
	Meta  Metadata // Meta is filled by the input and holds metadata that will be associated to the records parsed from Bytes
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

// InputStats contains statistics about the input component,
// ready for export to the metric client and to print debug info.
type InputStats struct {
	NumProcessedLines int64
	CustomStats       map[string]string
	Metrics           MetricsBag
}

// FilterStats contains statistics about the filter components,
// ready for export to the metric client and to print debug info
type FilterStats struct {
	NumProcessedLines int64
	NumFilteredLines  int64
	Metrics           MetricsBag
}

// ModifierStats...
type ModifierStats struct {
	Metrics MetricsBag
}

// OutputStats contains statistics about the output component,
// ready for export to the metric client and to print debug info
type OutputStats struct {
	NumProcessedLines int64
	NumErrorLines     int64
	CustomStats       map[string]string
	Metrics           MetricsBag
}

// UploadStats contains statistics about the upload component,
// ready for export to the metric client and to print debug info
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

// Filter represents a data filter; a filter is a function that processes
// records. A filter can discard, transform, forward and even create records.
type Filter interface {
	// Process processes a single Record, and then optionally sends it to
	// next filter in the chain.
	// Process might mutate the Record, adding/modifying/removing fields,
	// and might decide to throw it away, or pass it to next filter in chain
	// by calling the next() function. In some cases, a filter might generate
	// multiple Record in output, by calling next() multiple times.
	// next() is guaranteed to be non-nil; for the last filter of the chain,
	// it points to a function that wraps up the filtering chain and sends
	// the Record to the output.
	Process(l Record, next func(Record))

	// Stats returns stats about the filter
	Stats() FilterStats
}

type Modifier interface {
	Process(l Record)

	// Stats returns stats about the filter
	Stats() ModifierStats
}

// Output is the final end of a topology, it process the records that have
// reached the end of the filter chain and performs the final action (storing,
// sending through the wire, counting, etc.)
type Output interface {
	// Run processes the OutputRecord data coming through a channel.
	// Run must block until in channel has been closed and it has processed
	// all records.
	// It can send filenames via upch, they will be handled by an Upload if one
	// is present in the topology.
	// TODO: since Run must be blocking, it could return an error, useful
	// for the topology to acknowledge the correct processing if nil, or
	// end the whole topology in case non-nil.
	Run(in <-chan OutputRecord, upch chan<- string) error

	// Stats returns stats about the output.
	Stats() OutputStats

	// CanShards returns true if this output supports sharding.
	CanShard() bool
}

// OutputRecord is the data structure sent to baker output components.
//
// It represents a Record in two possibile formats:
//   * a list of pre-parsed fields, extracted from the record (as string).
//     This is useful when the output only cares about specific fields and does
//     not need the full record.
//   * the whole record, as processed and possibly modified by baker filters (as []byte).
//
// Fields sent to the output are described in the topology. This was designed
// such as an output can work in different modes, by processing different
// fields under the control of the user. Some fields might be required, and
// this validation should be performed by the Output itself. The topology can
// also declare no fields in which case, the Fields slice will be empty.
//
// Record is non-nil only if the output declares itself as a raw output (see
// OutputDesc.Raw). This is done for performance reasons, as recreating the
// whole record requires allocations and memory copies, and is not always
// required.
type OutputRecord struct {
	Fields []string // Fields are the fields sent to a Baker output.
	Record []byte   // Record is the data representation of a Record (obtained with Record.ToText())
}

// Upload uploads files created by the topology output to a configured location.
type Upload interface {
	// Run processes the output result as it comes through the channel.
	// Run must block forever
	// upch will receive filenames that Output wants to see uploaded.
	Run(upch <-chan string) error

	// Stop forces the upload to stop as cleanly as possible, which usually
	// means to finish up all the existing downloads.
	Stop()

	// Stats returns stats about the upload process
	Stats() UploadStats
}
