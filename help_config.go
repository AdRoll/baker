package baker

import (
	"fmt"
	"reflect"
)

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
