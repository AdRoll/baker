package baker

import (
	"fmt"
	"io"
	"reflect"
	"strings"
)

// GenerateTextHelp generates non-formatted textual help for a Baker
// component from its description structure, into w.
func GenerateTextHelp(w io.Writer, desc interface{}) error {
	if desc == nil {
		return fmt.Errorf("can't generate text help for a nil interface")
	}

	if reflect.TypeOf(desc).Kind() == reflect.Ptr {
		// dereference pointer
		desc = reflect.ValueOf(desc).Elem().Interface()
	}

	switch d := desc.(type) {
	case InputDesc:
		doc, err := newInputDoc(d)
		if err != nil {
			return err
		}
		genInputText(w, doc)
	case FilterDesc:
		doc, err := newFilterDoc(d)
		if err != nil {
			return err
		}
		genFilterText(w, doc)
	case OutputDesc:
		doc, err := newOutputDoc(d)
		if err != nil {
			return err
		}
		genOutputText(w, doc)
	case UploadDesc:
		doc, err := newUploadDoc(d)
		if err != nil {
			return err
		}
		genUploadText(w, doc)
	case MetricsDesc:
		doc, err := newMetricsDoc(d)
		if err != nil {
			return err
		}
		genMetricsText(w, doc)
	default:
		return fmt.Errorf("can't generate help, unsupported type %T", desc)
	}

	return nil
}

const (
	helpTextHdrSfmt = "%-18s | %-18s | %-18s | %-8s | |"
	helpTextSep     = "----------------------------------------------------------------------------------------------------"
)

func genInputText(w io.Writer, doc inputDoc) {
	fmt.Fprintf(w, "=============================================\n")
	fmt.Fprintf(w, "Input: %s\n", doc.name)
	fmt.Fprintf(w, "=============================================\n")
	fmt.Fprintf(w, doc.help)

	if len(doc.keys) == 0 {
		fmt.Fprintf(w, "\n(no configuration available)\n\n")
	} else {
		fmt.Fprintf(w, "\nKeys available in the [input.config] section:\n\n")
		genConfigKeysText(w, doc.keys)
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w)
}

func genFilterText(w io.Writer, doc filterDoc) {
	fmt.Fprintf(w, "=============================================\n")
	fmt.Fprintf(w, "Filter: %s\n", doc.name)
	fmt.Fprintf(w, "=============================================\n")
	fmt.Fprintf(w, doc.help)

	if len(doc.keys) == 0 {
		fmt.Fprintf(w, "\n(no configuration available)\n\n")
	} else {
		fmt.Fprintf(w, "\nKeys available in the [filter.config] section:\n\n")
		genConfigKeysText(w, doc.keys)
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w)
}

func genOutputText(w io.Writer, doc outputDoc) {
	const (
		rawString    = "This is a raw output, for each record it receives a buffer containing the serialized record, plus a list holding a set of fields ('output.fields' in TOML)."
		nonRawString = "This is a non-raw output, it doesn't receive whole records. Instead it receives a list of fields for each record ('output.fields' in TOML)."
	)

	fmt.Fprintf(w, "=============================================\n")
	fmt.Fprintf(w, "Output: %s\n", doc.name)
	fmt.Fprintf(w, "=============================================\n")
	fmt.Fprintf(w, doc.help)

	if doc.raw {
		fmt.Fprintln(w, rawString)
	} else {
		fmt.Fprintln(w, nonRawString)
	}

	if len(doc.keys) == 0 {
		fmt.Fprintf(w, "\n(no configuration available)\n\n")
	} else {
		fmt.Fprintf(w, "\nKeys available in the [output.config] section:\n\n")
		genConfigKeysText(w, doc.keys)
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w)
}

func genUploadText(w io.Writer, doc uploadDoc) {
	fmt.Fprintf(w, "=============================================\n")
	fmt.Fprintf(w, "Upload: %s\n", doc.name)
	fmt.Fprintf(w, "=============================================\n")
	fmt.Fprintf(w, doc.help)

	if len(doc.keys) == 0 {
		fmt.Fprintf(w, "\n(no configuration available)\n\n")
	} else {
		fmt.Fprintf(w, "\nKeys available in the [upload.config] section:\n\n")
		genConfigKeysText(w, doc.keys)
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w)
}

func genMetricsText(w io.Writer, doc metricsDoc) {
	fmt.Fprintf(w, "=============================================\n")
	fmt.Fprintf(w, "Metrics: %s\n", doc.name)
	fmt.Fprintf(w, "=============================================\n")

	if len(doc.keys) == 0 {
		fmt.Fprintf(w, "\n(no configuration available)\n\n")
	} else {
		fmt.Fprintf(w, "\nKeys available in the [metrics.config] section:\n\n")
		genConfigKeysText(w, doc.keys)
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w)
}

func genConfigKeysText(w io.Writer, keys []helpConfigKey) {
	hpad := fmt.Sprintf(helpTextHdrSfmt, "", "", "", "")

	fmt.Fprintf(w, helpTextHdrSfmt, "Name", "Type", "Default", "Required")
	fmt.Fprintf(w, "Help\n%s\n", helpTextSep)

	for _, k := range keys {
		fmt.Fprintf(w, helpTextHdrSfmt, k.name, k.typ, k.def, fmt.Sprintf("%t", k.required))
		helpLines := strings.Split(wrapString(k.desc, 60), "\n")
		if len(helpLines) > 0 {
			fmt.Fprint(w, helpLines[0], "\n")
			for _, h := range helpLines[1:] {
				fmt.Fprint(w, hpad, "  ", h, "\n")
			}
		} else {
			fmt.Fprint(w, "\n")
		}
	}

	fmt.Fprint(w, helpTextSep, "\n")
}
