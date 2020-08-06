package baker

// Components holds the descriptions of all components one can use
// to build a topology.
type Components struct {
	Inputs  []InputDesc
	Filters []FilterDesc
	Outputs []OutputDesc
	Uploads []UploadDesc
	User    []UserDesc

	ShardingFuncs map[FieldIndex]ShardingFunc
	Validate      ValidationFunc
	CreateRecord  func() Record

	FieldByName func(string) (FieldIndex, bool)
	FieldName   func(FieldIndex) string
}

// ComponentParams holds the common configuration parameters passed to components of all kinds.
type ComponentParams struct {
	DecodedConfig  interface{}                     // decoded component-specific struct (from configuration file)
	CreateRecord   func() Record                   // factory function to create new empty records
	FieldByName    func(string) (FieldIndex, bool) // translates field names to Record indexes
	FieldName      func(FieldIndex) string         // returns the name of a field given its index in the Record
	ValidateRecord ValidationFunc                  // function to validate a record
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
	Name   string
	New    func(InputParams) (Input, error)
	Config interface{}
	Help   string
}

// FilterDesc describes a Filter component to the topology.
type FilterDesc struct {
	Name   string
	New    func(FilterParams) (Filter, error)
	Config interface{}
	Help   string
}

// OutputDesc describes an Output component to the topology.
type OutputDesc struct {
	Name   string
	New    func(OutputParams) (Output, error)
	Config interface{}
	Raw    bool
	Help   string
}

// UploadDesc describes an Upload component to the topology.
type UploadDesc struct {
	Name   string
	New    func(UploadParams) (Upload, error)
	Config interface{}
	Help   string
}

// UserDesc describes user-specific configuration sections.
type UserDesc struct {
	Name   string
	Config interface{}
}
