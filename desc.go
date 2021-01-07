package baker

// Components holds the descriptions of all components one can use
// to build a topology.
type Components struct {
	Inputs  []InputDesc  // Inputs represents the list of available inputs
	Filters []FilterDesc // Filters represents the list of available filters
	Outputs []OutputDesc // Outputs represents the list of available outputs
	Uploads []UploadDesc // Uploads represents the list of available uploads

	Metrics []MetricsDesc // Metrics represents the list of available metrics clients
	User    []UserDesc    // User represents the list of user-defined configurations

	ShardingFuncs map[FieldIndex]ShardingFunc // ShardingFuncs are functions to calculate sharding based on field index
	Validate      ValidationFunc              // Validate is the function used to validate a Record
	CreateRecord  func() Record               // CreateRecord creates a new record

	FieldByName func(string) (FieldIndex, bool) // FieldByName gets a field index by its name
	FieldNames  []string                        // FieldNames holds field names, indexed by their FieldIndex
}

// ComponentParams holds the common configuration parameters passed to components of all kinds.
type ComponentParams struct {
	DecodedConfig  interface{}                     // decoded component-specific struct (from configuration file)
	CreateRecord   func() Record                   // factory function to create new empty records
	FieldByName    func(string) (FieldIndex, bool) // translates field names to Record indexes
	FieldNames     []string                        // FieldNames holds field names, indexed by their FieldIndex
	ValidateRecord ValidationFunc                  // function to validate a record
	Metrics        MetricsClient                   // Metrics allows components to add code instrumentation and have metrics exported to the configured backend, if any?
}

// InputParams holds the parameters passed to Input constructor.
type InputParams struct {
	ComponentParams
}

// FilterParams holds the parameters passed to Filter constructor.
type FilterParams struct {
	ComponentParams
}

// OutputParams holds the parameters passed to Output constructor.
type OutputParams struct {
	ComponentParams
	Index  int          // tells the index of the output, in case multiple parallel output procs are used
	Fields []FieldIndex // fields of the record that will be send to the output
}

// UploadParams is the struct passed to the Upload constructor.
type UploadParams struct {
	ComponentParams
}

// A ShardingFunc calculates a sharding value for a record.
//
// Sharding functions are silent to errors in the specified fields. If a field
// is corrupt, they will probabily ignore it and still compute the best
// possible sharding value. Obviously a very corrupted field (eg: empty) could
// result into an uneven sharding.
type ShardingFunc func(Record) uint64

// ValidationFunc checks the validity of a record, returning true if it's
// valid. If a validation error is found it returns false and the index of
// the field that failed validation.
type ValidationFunc func(Record) (bool, FieldIndex)

// InputDesc describes an Input component to the topology.
type InputDesc struct {
	Name   string                           // Name of the input
	New    func(InputParams) (Input, error) // New is the constructor-like function called by the topology to create a new input
	Config interface{}                      // Config is the component configuration
	Help   string                           // Help string
}

// FilterDesc describes a Filter component to the topology.
type FilterDesc struct {
	Name   string                             // Name of the filter
	New    func(FilterParams) (Filter, error) // New is the constructor-like function called by the topology to create a new filter
	Config interface{}                        // Config is the component configuration
	Help   string                             // Help string
}

// OutputDesc describes an Output component to the topology.
type OutputDesc struct {
	Name   string                             // Name of the output
	New    func(OutputParams) (Output, error) // New is the constructor-like function called by the topology to create a new output
	Config interface{}                        // Config is the component configuration
	Raw    bool                               // Raw reports whether the output accepts a raw record
	Help   string                             // Help string
}

// UploadDesc describes an Upload component to the topology.
type UploadDesc struct {
	Name   string                             // Name of the upload component
	New    func(UploadParams) (Upload, error) // New is the constructor-like function called by the topology to create a new upload
	Config interface{}                        // Config is the component configuration
	Help   string                             // Help string
}

// MetricsDesc describes a Metrics interface to the topology.
type MetricsDesc struct {
	Name   string                                   // Name of the metrics interface
	Config interface{}                              // Config is the metrics client specific configuration
	New    func(interface{}) (MetricsClient, error) // New is the constructor-like function called by the topology to create a new metrics client
}

// UserDesc describes user-specific configuration sections.
type UserDesc struct {
	Name   string
	Config interface{}
}
