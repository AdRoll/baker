package baker

// Components holds the descriptions of all components one can use
// to build a topology.
type Components struct {
	Inputs  []InputDesc  // list of available inputs
	Filters []FilterDesc // list of available filters
	Outputs []OutputDesc // list of available outputs
	Uploads []UploadDesc // list of available uploads

	Metrics []MetricsDesc // list of available metrics clients
	User    []UserDesc    // list of user-defined configurations

	ShardingFuncs map[FieldIndex]ShardingFunc // functions to calculate sharding based on field index
	Validate      ValidationFunc              // function to use to validate a Records
	CreateRecord  func() Record               // create a new record

	FieldByName func(string) (FieldIndex, bool) // get a field index by its name
	FieldName   func(FieldIndex) string         // gets a field name by its index
}

// ComponentParams holds the common configuration parameters passed to components of all kinds.
type ComponentParams struct {
	DecodedConfig  interface{}                     // decoded component-specific struct (from configuration file)
	CreateRecord   func() Record                   // factory function to create new empty records
	FieldByName    func(string) (FieldIndex, bool) // translates field names to Record indexes
	FieldName      func(FieldIndex) string         // returns the name of a field given its index in the Record
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
//
// It has a name, a config object, a constructor function (New)
// and a help string
type InputDesc struct {
	Name   string
	New    func(InputParams) (Input, error)
	Config interface{}
	Help   string
}

// FilterDesc describes a Filter component to the topology.
//
// It has a name, a config object, a constructor function (New)
// and a help string
type FilterDesc struct {
	Name   string
	New    func(FilterParams) (Filter, error)
	Config interface{}
	Help   string
}

// OutputDesc describes an Output component to the topology.
//
// It has a name, a config object, a constructor function (New)
// and a help string. Raw defines whether the output accepts
// raw records
type OutputDesc struct {
	Name   string
	New    func(OutputParams) (Output, error)
	Config interface{}
	Raw    bool
	Help   string
}

// UploadDesc describes an Upload component to the topology.
//
// It has a name, a config object, a constructor function (New)
// and a help string
type UploadDesc struct {
	Name   string
	New    func(UploadParams) (Upload, error)
	Config interface{}
	Help   string
}

// MetricsDesc describes a Metrics interface to the topology.
type MetricsDesc struct {
	Name   string                                   // Name of the metrics interface
	Config interface{}                              // Config is the metrics client specific configuration
	New    func(interface{}) (MetricsClient, error) // Constructor
}

// UserDesc describes user-specific configuration sections.
type UserDesc struct {
	Name   string
	Config interface{}
}
