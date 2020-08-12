package baker

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"regexp"
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
	ChanSize      int
	DecodedConfig interface{}

	Config *toml.Primitive
	desc   *InputDesc
}

// ConfigFilterChain specifies the configuration for the whole fitler chain.
type ConfigFilterChain struct {
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
	Name          string
	Procs         int
	ChanSize      int
	RawChanSize   int
	Sharding      string
	Fields        []string
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
	Datadog       bool
	DatadogPrefix string   `toml:"datadog_prefix"`
	DatadogHost   string   `toml:"datadog_host"`
	DatadogTags   []string `toml:"datadog_tags"`
	// DatadogSendLogs indicates whether baker log entries are forwarded as
	// statsd events to the datadog-agent listening at DatadogHost.
	DatadogSendLogs bool `toml:"datadog_send_logs"`

	DontValidateFields bool `toml:"dont_validate_fields"`
}

// A Config specifies the configuration for a topology.
type Config struct {
	Input       ConfigInput
	FilterChain ConfigFilterChain
	Filter      []ConfigFilter
	Output      ConfigOutput
	Upload      ConfigUpload
	General     ConfigGeneral
	User        []ConfigUser
	CSV         ConfigCSV

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
	s += fmt.Sprintf("Output:{Name:%s, Procs:%d, ChanSize:%d, RawChanSize:%d, Sharding:%s, Fields:[%s]} ", c.Output.Name, c.Output.Procs, c.Output.ChanSize, c.Output.RawChanSize, c.Output.Sharding, strings.Join(c.Output.Fields, ","))
	s += fmt.Sprintf("Upload:{Name:%s}", c.Upload.Name)
	return s
}

func (c *Config) fillDefaults() error {
	c.Input.fillDefaults()
	c.FilterChain.fillDefaults()
	c.Output.fillDefaults()
	c.Upload.fillDefaults()
	c.General.fillDefaults()
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
	if c.RawChanSize == 0 {
		c.RawChanSize = 16384
	}
	if c.Procs == 0 {
		c.Procs = 32
	}
}

func (c *ConfigUpload) fillDefaults() {}

func (c *ConfigGeneral) fillDefaults() {
	if c.DatadogPrefix == "" {
		c.DatadogPrefix = "baker."
	}
	if c.DatadogHost == "" {
		c.DatadogHost = "127.0.0.1:8125"
	}
}

// cloneConfig clones a configuration object.
func cloneConfig(i interface{}) interface{} {
	return reflect.New(reflect.ValueOf(i).Elem().Type()).Interface()
}

var envVarRegxp = regexp.MustCompile(`\${(\w+)}`)

// replaceEnvVars replaces any string in the format ${VALUE} with the corresponding
// $VALUE environment variable. It returns error if the variable can't be found or
// if any error occurs while reading from the input reader.
func replaceEnvVars(f io.Reader) (io.Reader, error) {
	buf, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("Error reading input: %v", err)
	}

	for _, match := range envVarRegxp.FindAllSubmatch(buf, -1) {
		v := os.Getenv(string(match[1]))
		if v != "" {
			buf = bytes.Replace(buf, match[0], []byte(v), 1)
		}
	}

	return bytes.NewReader(buf), nil
}

// NewConfigFromToml creates a Config from a reader reading from a TOML
// configuration. comp describes all the existing components.
func NewConfigFromToml(f io.Reader, comp Components) (*Config, error) {
	f, err := replaceEnvVars(f)
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

	// Copy custom configuration structure, to prepare for re-reading
	cfg.Input.DecodedConfig = cfg.Input.desc.Config
	if cfg.Input.Config != nil {
		if err := md.PrimitiveDecode(*cfg.Input.Config, cfg.Input.DecodedConfig); err != nil {
			return nil, fmt.Errorf("error parsing input config: %v", err)
		}
	}

	for idx := range cfg.Filter {
		// Clone the configuration object to allow the use of multiple instances of the same filter
		cfg.Filter[idx].DecodedConfig = cloneConfig(cfg.Filter[idx].desc.Config)
		if cfg.Filter[idx].Config != nil {
			if err := md.PrimitiveDecode(*cfg.Filter[idx].Config, cfg.Filter[idx].DecodedConfig); err != nil {
				return nil, fmt.Errorf("error parsing filter config: %v", err)
			}
		}
	}

	cfg.Output.DecodedConfig = cfg.Output.desc.Config
	if cfg.Output.Config != nil {
		if err := md.PrimitiveDecode(*cfg.Output.Config, cfg.Output.DecodedConfig); err != nil {
			return nil, fmt.Errorf("error parsing output config: %v", err)
		}
	}

	if cfg.Upload.Name != "" {
		cfg.Upload.DecodedConfig = cfg.Upload.desc.Config
		if cfg.Upload.Config != nil {
			if err := md.PrimitiveDecode(*cfg.Upload.Config, cfg.Upload.DecodedConfig); err != nil {
				return nil, fmt.Errorf("error parsing upload config: %v", err)
			}
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

	// Copy pluggable functions
	cfg.shardingFuncs = comp.ShardingFuncs
	cfg.validate = comp.Validate
	cfg.createRecord = comp.CreateRecord
	cfg.fieldByName = comp.FieldByName
	cfg.fieldName = comp.FieldName

	// Fill-in with missing defaults
	return &cfg, cfg.fillDefaults()
}
