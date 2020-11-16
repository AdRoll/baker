package baker

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"unicode"

	"github.com/rasky/toml"
)

// The configuration for the topology is parsed from TOML format.
// Each of Input, Filter, Output have a separate section (table)
// in the file containing a key called "Name" that specifies
// which component will be used in the pipeline, plus a couple
// of other generic keys.
//
// Then, there are sub-tables called [input.config], [filter.config],
// [output.config] that contains configuration specific of each
// component, and directly map to the *Config structure in the code,
// as specified in the components descriptions (see input.AllInputs(),
// filter.AllFilters(), output.AllOutputs(), upload.AllUploads()).
//
// To match this with the TOML parser, we need to use deferred parsing,
// because initially we don't know which component has been chosen and
// thus we cannot provide to the toml decoder the correct config instances.
// toml.Primitive is how deferred parsing is implemented (it's the
// equivalent of encoding/json.RawMessage). Then, in the second step,
// the DecodedConfig structure is populated correctly with the component
// configuration.

// ConfigInput specifies the configuration for the input component.
type ConfigInput struct {
	Name          string
	ChanSize      int // ChanSize represents the size of the channel to send records from the input to the filters, the default value is 1024
	DecodedConfig interface{}

	Config *toml.Primitive
	desc   *InputDesc
}

// ConfigFilterChain specifies the configuration for the whole fitler chain.
type ConfigFilterChain struct {
	// Procs specifies the number of baker filters running concurrently.
	// When set to a value greater than 1, filtering may be faster but
	// record ordering is not guaranteed anymore.
	// The default value is 16
	Procs int
}

// ConfigFilter specifies the configuration for a single filter component.
type ConfigFilter struct {
	Name          string
	DecodedConfig interface{}

	Config *toml.Primitive
	desc   *FilterDesc
}

// ConfigOutput specifies the configuration for the output component.
type ConfigOutput struct {
	Name string
	// Procs defines the number of baker outputs running concurrently.
	// Only set Procs to a value greater than 1 if the output is concurrent safe.
	Procs         int
	ChanSize      int      // ChanSize represents the size of the channel to send records to the ouput component(s), the default value is 16384
	Sharding      string   // Sharding is the name of the field used for sharding
	Fields        []string // Fields holds the name of the record fields the output receives
	DecodedConfig interface{}

	Config *toml.Primitive
	desc   *OutputDesc
}

// ConfigUpload specifies the configuration for the upload component.
type ConfigUpload struct {
	Name          string
	DecodedConfig interface{}

	Config *toml.Primitive
	desc   *UploadDesc
}

// A ConfigUser defines a user-specific configuration entry.
type ConfigUser struct {
	Name   string
	Config *toml.Primitive
}

// ConfigCSV defines configuration for CSV records
type ConfigCSV struct {
	// FieldSeparator defines the fields separator used in the records
	FieldSeparator string `toml:"field_separator"`
}

// A ConfigGeneral specifies general configuration for the whole topology.
type ConfigGeneral struct {
	// DontValidateFields reports whether records validation is skipped (by not calling Components.Validate)
	DontValidateFields bool `toml:"dont_validate_fields"`
}

// ConfigMetrics holds metrics configuration.
type ConfigMetrics struct {
	Name          string
	DecodedConfig interface{}

	Config *toml.Primitive
	desc   *MetricsDesc
}

// ConfigFields specifies names for records fields. In addition of being a list
// of names, the position of each name in the slice also indicates the FieldIndex
// for that name. In other words, if Names[0] = "address", then a FieldIndex of
// 0 is that field, and "address" is the name of that field.
type ConfigFields struct {
	Names []string
}

// A Config specifies the configuration for a topology.
type Config struct {
	Input       ConfigInput
	FilterChain ConfigFilterChain
	Filter      []ConfigFilter
	Output      ConfigOutput
	Upload      ConfigUpload

	General ConfigGeneral
	Fields  ConfigFields
	Metrics ConfigMetrics
	CSV     ConfigCSV
	User    []ConfigUser

	shardingFuncs map[FieldIndex]ShardingFunc
	validate      ValidationFunc
	createRecord  func() Record

	fieldByName func(string) (FieldIndex, bool)
	fieldName   func(FieldIndex) string
}

// String returns a string representation of the exported fields of c.
func (c *Config) String() string {
	s := fmt.Sprintf("Input:{Name:%s, ChanSize:%d} ", c.Input.Name, c.Input.ChanSize)
	s += fmt.Sprintf("FilterChain:{Procs:%d} ", c.FilterChain.Procs)
	for i, f := range c.Filter {
		s += fmt.Sprintf("Filter-%d:{Name:%s} ", i, f.Name)
	}
	s += fmt.Sprintf("Output:{Name:%s, Procs:%d, ChanSize:%d, Sharding:%s, Fields:[%s]} ", c.Output.Name, c.Output.Procs, c.Output.ChanSize, c.Output.Sharding, strings.Join(c.Output.Fields, ","))
	s += fmt.Sprintf("Upload:{Name:%s}", c.Upload.Name)
	return s
}

func (c *Config) fillDefaults() error {
	c.Input.fillDefaults()
	c.FilterChain.fillDefaults()
	c.Output.fillDefaults()
	c.Upload.fillDefaults()
	if err := c.fillCreateRecordDefault(); err != nil {
		return err
	}
	return nil
}

func (c *Config) fillCreateRecordDefault() error {
	if c.createRecord == nil {
		fieldSeparator := DefaultLogLineFieldSeparator
		if c.CSV.FieldSeparator != "" {
			sep := []rune(c.CSV.FieldSeparator)
			if len(sep) != 1 || sep[0] > unicode.MaxASCII {
				return fmt.Errorf("Separator must be a 1-byte string or hex char")
			}
			fieldSeparator = byte(sep[0])
		}
		// For now, leave Logline as the default
		c.createRecord = func() Record {
			return &LogLine{
				FieldSeparator: fieldSeparator,
			}
		}
	}
	return nil
}

func (c *ConfigInput) fillDefaults() {
	if c.ChanSize == 0 {
		c.ChanSize = 1024
	}
}

func (c *ConfigFilterChain) fillDefaults() {
	if c.Procs == 0 {
		c.Procs = 16
	}
}

func (c *ConfigOutput) fillDefaults() {
	if c.ChanSize == 0 {
		c.ChanSize = 16384
	}
	if c.Procs == 0 {
		c.Procs = 32
	}
}

func (c *ConfigUpload) fillDefaults() {}

// cloneConfig clones a configuration object.
func cloneConfig(i interface{}) interface{} {
	return reflect.New(reflect.ValueOf(i).Elem().Type()).Interface()
}

// replaceEnvVars replaces any string in the format ${VALUE} or $VALUE with the corresponding
// $VALUE environment variable
func replaceEnvVars(f io.Reader, mapper func(string) string) (io.Reader, error) {
	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(f)
	if err != nil {
		return nil, fmt.Errorf("Error reading input: %v", err)
	}

	return strings.NewReader(os.Expand(buf.String(), mapper)), nil
}

func decodeAndCheckConfig(md toml.MetaData, compCfg interface{}) error {
	var (
		cfg  *toml.Primitive // config
		dcfg interface{}     // decoded config
		name string          // component name
		typ  string          // component type
	)

	switch t := compCfg.(type) {
	case ConfigInput:
		cfg, dcfg = t.Config, t.DecodedConfig
		name, typ = t.Name, "input"
	case ConfigFilter:
		cfg, dcfg = t.Config, t.DecodedConfig
		name, typ = t.Name, "filter"
	case ConfigOutput:
		cfg, dcfg = t.Config, t.DecodedConfig
		name, typ = t.Name, "output"
	case ConfigUpload:
		cfg, dcfg = t.Config, t.DecodedConfig
		name, typ = t.Name, "upload"
	case ConfigMetrics:
		cfg, dcfg = t.Config, t.DecodedConfig
		name, typ = t.Name, "metrics"
	default:
		panic(fmt.Sprintf("unexpected type %#v", cfg))
	}

	if cfg == nil {
		// No config section was given in the TOML, so we create a pointer to
		// the zero value of our config struct to check required field
		tcfg := reflect.TypeOf(dcfg).Elem()
		dcfg = reflect.New(tcfg).Interface()
	} else {
		if err := md.PrimitiveDecode(*cfg, dcfg); err != nil {
			return fmt.Errorf("%s %q: error parsing config: %v", typ, name, err)
		}
	}

	if req := CheckRequiredFields(dcfg); req != "" {
		return fmt.Errorf("%s %q: %w", typ, name, ErrorRequiredField{req})
	}

	return nil
}

// NewConfigFromToml creates a Config from a reader reading from a TOML
// configuration. comp describes all the existing components.
func NewConfigFromToml(f io.Reader, comp Components) (*Config, error) {
	f, err := replaceEnvVars(f, os.Getenv)
	if err != nil {
		return nil, fmt.Errorf("Can't replace config with env vars: %v", err)
	}

	// Parse che configuration. Part of the configuration will be
	// captured as toml.Primitive for deferred parsing (see comment
	// at top of the file)
	cfg := Config{}
	md, err := toml.DecodeReader(f, &cfg)
	if err != nil {
		return nil, fmt.Errorf("error parsing topology: %v", err)
	}

	// We now go through inputs, filters and outputs, and match the names
	// to the actual object descriptions provided by each respective package.
	// Through the description, we also acquire an instance to the actual
	// custom configuration structure, that will be filled later.
	for _, inp := range comp.Inputs {
		if strings.EqualFold(inp.Name, cfg.Input.Name) {
			cfg.Input.desc = &inp
			break
		}
	}
	if cfg.Input.desc == nil {
		return nil, fmt.Errorf("input does not exist: %q", cfg.Input.Name)
	}

	for idx := range cfg.Filter {
		cfgfil := &cfg.Filter[idx]
		for _, fil := range comp.Filters {
			if strings.EqualFold(fil.Name, cfgfil.Name) {
				cfgfil.desc = &fil
				break
			}
		}
		if cfgfil.desc == nil {
			return nil, fmt.Errorf("filter does not exist: %q", cfgfil.Name)
		}
	}

	for _, out := range comp.Outputs {
		if strings.EqualFold(out.Name, cfg.Output.Name) {
			cfg.Output.desc = &out
			break
		}
	}
	if cfg.Output.desc == nil {
		return nil, fmt.Errorf("output does not exist: %q", cfg.Output.Name)
	}

	// Upload can be empty
	for _, upl := range comp.Uploads {
		if strings.EqualFold(upl.Name, cfg.Upload.Name) {
			cfg.Upload.desc = &upl
			break
		}
	}

	if cfg.Metrics.Name != "" {
		for _, mtr := range comp.Metrics {
			if strings.EqualFold(mtr.Name, cfg.Metrics.Name) {
				cfg.Metrics.desc = &mtr
				break
			}
		}
		if cfg.Metrics.desc == nil {
			return nil, fmt.Errorf("metrics does not exist: %q", cfg.Metrics.Name)
		}
	}

	// Copy custom configuration structure, to prepare for re-reading
	cfg.Input.DecodedConfig = cfg.Input.desc.Config
	if err := decodeAndCheckConfig(md, cfg.Input); err != nil {
		return nil, err
	}

	for idx := range cfg.Filter {
		// Clone the configuration object to allow the use of multiple instances of the same filter
		cfg.Filter[idx].DecodedConfig = cloneConfig(cfg.Filter[idx].desc.Config)
		if err := decodeAndCheckConfig(md, cfg.Filter[idx]); err != nil {
			return nil, err
		}
	}

	cfg.Output.DecodedConfig = cfg.Output.desc.Config
	if err := decodeAndCheckConfig(md, cfg.Output); err != nil {
		return nil, err
	}

	if cfg.Upload.Name != "" {
		cfg.Upload.DecodedConfig = cfg.Upload.desc.Config
		if err := decodeAndCheckConfig(md, cfg.Upload); err != nil {
			return nil, err
		}
	}

	if cfg.Metrics.Name != "" {
		cfg.Metrics.DecodedConfig = cfg.Metrics.desc.Config
		if err := decodeAndCheckConfig(md, cfg.Metrics); err != nil {
			return nil, err
		}
	}

	// Decode user-specific configuration entries.
	for _, cfgUser := range cfg.User {
		found := false
		for idx := range comp.User {
			if strings.EqualFold(cfgUser.Name, comp.User[idx].Name) {
				found = true
				if err := md.PrimitiveDecode(*cfgUser.Config, comp.User[idx].Config); err != nil {
					return nil, fmt.Errorf("error parsing user config: %v", err)
				}
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("user configuration does not exist: %q", cfgUser.Name)
		}
	}

	// Abort if there's any unknown key in the configuration file
	if keys := md.Undecoded(); len(keys) > 0 {
		return nil, fmt.Errorf("invalid keys in configuration file: %v", keys)
	}

	if err := assignFieldMapping(&cfg, comp); err != nil {
		return nil, err
	}

	// Copy pluggable functions
	cfg.shardingFuncs = comp.ShardingFuncs
	cfg.validate = comp.Validate
	cfg.createRecord = comp.CreateRecord

	// Fill-in with missing defaults
	return &cfg, cfg.fillDefaults()
}

// hasConfig returns true if the underlying structure has at least one field.
func hasConfig(cfg interface{}) bool {
	tf := reflect.TypeOf(cfg).Elem()
	return tf.NumField() != 0
}

// assignFieldMapping verifies that field mapping has been set once, but only
// once (either in cfg or comp). Then if that is the case, assignFieldMapping
// sets both fieldByName and fieldName in cfg.
func assignFieldMapping(cfg *Config, comp Components) error {
	cfgOk := len(cfg.Fields.Names) != 0
	compOk := comp.FieldByName != nil && comp.FieldName != nil

	if (comp.FieldByName == nil) != (comp.FieldName == nil) {
		return fmt.Errorf("FieldByName and FieldName must be either both set or both unset")
	}

	// First, get inconsistent cases out of the way.
	if !cfgOk && !compOk {
		return fmt.Errorf("field indexes/names have not been set")
	}

	if cfgOk && compOk {
		return fmt.Errorf("field indexes/names can't both be set in TOML and in Components")
	}

	if compOk {
		// Ok, mapping has been set from Components.
		cfg.fieldByName = comp.FieldByName
		cfg.fieldName = comp.FieldName
		return nil
	}

	// Mapping has been set from Config, create both closures and assign them.
	m := make(map[string]FieldIndex, len(cfg.Fields.Names))
	for f, s := range cfg.Fields.Names {
		_, ok := m[s]
		if ok {
			return fmt.Errorf("duplicated field name %q", s)
		}
		m[s] = FieldIndex(f)
	}

	cfg.fieldByName = func(name string) (FieldIndex, bool) {
		f, ok := m[name]
		return f, ok
	}
	cfg.fieldName = func(fidx FieldIndex) string {
		return cfg.Fields.Names[fidx]
	}

	return nil
}

// RequiredFields returns the names of the underlying configuration structure
// fields which are tagged as required. To tag a field as being required, a
// "required" struct struct tag must be present and set to true.
//
// RequiredFields doesn't support struct embedding other structs.
func RequiredFields(cfg interface{}) []string {
	var fields []string

	tf := reflect.TypeOf(cfg).Elem()
	for i := 0; i < tf.NumField(); i++ {
		field := tf.Field(i)

		req := field.Tag.Get("required")
		if req != "true" {
			continue
		}

		fields = append(fields, field.Name)
	}

	return fields
}

// CheckRequiredFields checks that all fields that are tagged as required in
// cfg's type have actually been set to a value other than the field type zero
// value.  If not CheckRequiredFields returns the name of the first required
// field that is not set, or, it returns an empty string if all required fields
// are set of the struct doesn't have any required fields (or any fields at all).
//
// CheckRequiredFields doesn't support struct embedding other structs.
func CheckRequiredFields(cfg interface{}) string {
	fields := RequiredFields(cfg)

	for _, name := range fields {
		rv := reflect.ValueOf(cfg).Elem()
		fv := rv.FieldByName(name)
		if fv.IsZero() {
			return name
		}
	}

	return ""
}

// ErrorRequiredField describes the absence of a required field
// in a component configuration.
type ErrorRequiredField struct {
	Field string // Field is the name of the missing field
}

func (e ErrorRequiredField) Error() string {
	return fmt.Sprintf("%q is a required field", e.Field)
}
