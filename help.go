package baker

import (
	"fmt"
	"io"
	"reflect"
	"strings"
)

// HelpFormat represents the possible formats for baker help.
type HelpFormat int

const (
	// HelpFormatRaw is for raw-formatted help.
	HelpFormatRaw HelpFormat = iota

	// HelpFormatMarkdown is for markdown formatted help.
	HelpFormatMarkdown
)

// PrintHelp prints the help message for the given component, identified by its name.
// When name is '*' it shows the help messages for all components.
//
// The help message includes the component's description as well as the help messages
// for all component's configuration keys.
//
// An example of usage is:
//
//     var flagPrintHelp = flag.String("help", "", "show help for a `component` ('*' for all)")
//     flag.Parse()
//     comp := baker.Components{ /* add all your baker components here */ }
//     PrintHelp(os.Stderr, *flagPrintHelp, comp)
//
// Help output example:
//
//     $ ./baker-bin -help TCP
//
//     =============================================
//     Input: TCP
//     =============================================
//     This input relies on a TCP connection to receive records in the usual format
//     Configure it with a host and port that you want to accept connection from.
//     By default it listens on port 6000 for any connection
//     It never exits.
//
//     Keys available in the [input.config] section:
//
//     Name               | Type               | Default                    | Help
//     ----------------------------------------------------------------------------------------------------
//     Listener           | string             |                            | Host:Port to bind to
//     ----------------------------------------------------------------------------------------------------
func PrintHelp(w io.Writer, name string, comp Components, format HelpFormat) error {
	dumpall := name == "*"

	generateHelp := GenerateTextHelp
	if format == HelpFormatMarkdown {
		generateHelp = GenerateMarkdownHelp
	}

	for _, inp := range comp.Inputs {
		if strings.EqualFold(inp.Name, name) || dumpall {
			if err := generateHelp(w, inp); err != nil {
				return fmt.Errorf("can't print help for %q input: %v", inp.Name, err)
			}
			if !dumpall {
				return nil
			}
		}
	}

	for _, fil := range comp.Filters {
		if strings.EqualFold(fil.Name, name) || dumpall {
			if err := generateHelp(w, fil); err != nil {
				return fmt.Errorf("can't print help for %q filter: %v", fil.Name, err)
			}
			if !dumpall {
				return nil
			}
		}
	}

	for _, out := range comp.Outputs {
		if strings.EqualFold(out.Name, name) || dumpall {
			if strings.EqualFold(out.Name, name) || dumpall {
				if err := generateHelp(w, out); err != nil {
					return fmt.Errorf("can't print help for %q output: %v", out.Name, err)
				}
				if !dumpall {
					return nil
				}
			}
		}
	}

	for _, upl := range comp.Uploads {
		if strings.EqualFold(upl.Name, name) || dumpall {
			if strings.EqualFold(upl.Name, name) || dumpall {
				if err := generateHelp(w, upl); err != nil {
					return fmt.Errorf("can't print help for %q upload: %v", upl.Name, err)
				}
				if !dumpall {
					return nil
				}
			}
		}
	}

	if !dumpall {
		return fmt.Errorf("component not found: %s", name)
	}

	return nil
}

type baseDoc struct {
	name string          // component name
	help string          // general help string
	keys []helpConfigKey // configuration keys
}

type inputDoc struct{ baseDoc }
type filterDoc struct{ baseDoc }
type uploadDoc struct{ baseDoc }

type outputDoc struct {
	baseDoc
	raw bool // raw output?
}

type metricsDoc struct {
	name string          // component name
	keys []helpConfigKey // configuration keys
}

func newInputDoc(desc InputDesc) (inputDoc, error) {
	doc := inputDoc{
		baseDoc{
			name: desc.Name,
			help: desc.Help,
		},
	}

	var err error

	doc.keys, err = configKeysFromStruct(desc.Config)
	if err != nil {
		return doc, fmt.Errorf("input %q: %v", desc.Name, err)
	}

	return doc, nil
}

func newFilterDoc(desc FilterDesc) (filterDoc, error) {
	doc := filterDoc{
		baseDoc{
			name: desc.Name,
			help: desc.Help,
		},
	}

	var err error

	doc.keys, err = configKeysFromStruct(desc.Config)
	if err != nil {
		return doc, fmt.Errorf("filter %q: %v", desc.Name, err)
	}

	return doc, nil
}

func newOutputDoc(desc OutputDesc) (outputDoc, error) {
	doc := outputDoc{
		raw: desc.Raw,
		baseDoc: baseDoc{
			name: desc.Name,
			help: desc.Help,
		},
	}

	var err error

	doc.keys, err = configKeysFromStruct(desc.Config)
	if err != nil {
		return doc, fmt.Errorf("output %q: %v", desc.Name, err)
	}

	return doc, nil
}

func newUploadDoc(desc UploadDesc) (uploadDoc, error) {
	doc := uploadDoc{
		baseDoc{
			name: desc.Name,
			help: desc.Help,
		},
	}

	var err error

	doc.keys, err = configKeysFromStruct(desc.Config)
	if err != nil {
		return doc, fmt.Errorf("upload %q: %v", desc.Name, err)
	}

	return doc, nil
}

func newMetricsDoc(desc MetricsDesc) (metricsDoc, error) {
	doc := metricsDoc{
		name: desc.Name,
	}

	var err error

	doc.keys, err = configKeysFromStruct(desc.Config)
	if err != nil {
		return doc, fmt.Errorf("metrics %q: %v", desc.Name, err)
	}

	return doc, nil
}

type helpConfigKey struct {
	name     string // config key name
	typ      string // config key type
	def      string // default value
	required bool
	desc     string
}

func configKeysFromStruct(cfg interface{}) ([]helpConfigKey, error) {
	var keys []helpConfigKey

	tf := reflect.TypeOf(cfg).Elem()
	for i := 0; i < tf.NumField(); i++ {
		f := tf.Field(i)

		// skip unexported fields
		if f.PkgPath != "" && !f.Anonymous {
			continue
		}

		key, err := newHelpConfigKeyFromField(f)
		if err != nil {
			return nil, fmt.Errorf("error at exported key %d: %v", i, err)
		}
		keys = append(keys, key)
	}

	return keys, nil
}

func newHelpConfigKeyFromField(f reflect.StructField) (helpConfigKey, error) {
	h := helpConfigKey{
		name:     f.Name,
		desc:     f.Tag.Get("help"),
		def:      f.Tag.Get("default"),
		required: f.Tag.Get("required") == "true",
	}

	switch f.Type.Kind() {
	case reflect.Int:
		h.typ = "int"
	case reflect.String:
		h.typ = "string"
		h.def = `"` + h.def + `"`
	case reflect.Slice:
		switch f.Type.Elem().Kind() {
		case reflect.String:
			h.typ = "array of strings"
		case reflect.Int:
			h.typ = "array of ints"
		default:
			return h, fmt.Errorf("config key %q: unsupported type array of %s", f.Type.Name(), f.Type.Elem())
		}
	case reflect.Int64:
		if f.Type.Name() == "Duration" {
			h.typ = "duration"
		} else {
			h.typ = "int"
		}
	case reflect.Bool:
		h.typ = "bool"
	default:
		return h, fmt.Errorf("config key %q: unsupported type", f.Type.Name())
	}

	return h, nil
}
