package baker

import (
	"fmt"
	"io"
	"reflect"
	"runtime"
	"strings"
	"syscall"
	"unsafe"

	"github.com/charmbracelet/glamour"
)

// RenderHelpMarkdown prints markdown formatted help for a single component or
// for all of them (with name = '*'), and renders it so that it can be
// printed on a terminal.
func RenderHelpMarkdown(w io.Writer, name string, comp Components) error {
	r, _ := glamour.NewTermRenderer(
		// detect background color and pick either the default dark or light theme
		glamour.WithAutoStyle(),
		// wrap output at specific width
		glamour.WithWordWrap(int(terminalWidth())),
	)

	if err := PrintHelp(r, name, comp, HelpFormatMarkdown); err != nil {
		return err
	}

	r.Close()
	_, err := io.Copy(w, r)
	return err
}

func terminalWidth() uint {
	const (
		maxWidth     = 140 // don't go over 140 chars anyway
		defaultWidth = 110 // in case we can't get the terminal width
	)

	var w uint

	defer func() {
		if err := recover(); err != nil {
			w = defaultWidth
		}
	}()

	if runtime.GOOS == "windows" {
		// On windows assume 120 character wide terminal since the subsequent
		// method only works on nix systems.
		return 120
	}

	ws := &struct{ Row, Col, Xpixel, Ypixel uint16 }{}
	retCode, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(syscall.Stdout),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(ws)))

	if int(retCode) == -1 {
		panic(errno)
	}

	if ws.Col > maxWidth {
		return maxWidth
	}

	w = uint(ws.Col)
	return w
}

// GenerateMarkdownHelp generates markdown-formatted textual help for a Baker
// component from its description structure. Markdown is written into w.
func GenerateMarkdownHelp(w io.Writer, desc interface{}) error {
	if desc == nil {
		return fmt.Errorf("can't generate markdown help for a nil interface")
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
		genInputMarkdown(w, doc)
	case FilterDesc:
		doc, err := newFilterDoc(d)
		if err != nil {
			return err
		}
		genFilterMarkdown(w, doc)
	case OutputDesc:
		doc, err := newOutputDoc(d)
		if err != nil {
			return err
		}
		genOutputMarkdown(w, doc)
	case UploadDesc:
		doc, err := newUploadDoc(d)
		if err != nil {
			return err
		}
		genUploadMarkdown(w, doc)
	case MetricsDesc:
		doc, err := newMetricsDoc(d)
		if err != nil {
			return err
		}
		genMetricsMarkdown(w, doc)
	default:
		return fmt.Errorf("can't generate markdown help, unsupported type %T", desc)
	}

	return nil
}

func breakAfterDots(s string) string {
	r := strings.NewReplacer(
		".", ".  \n",
		"!", "!  \n",
		"?", "?  \n",
	)
	return r.Replace(s)
}

func genInputMarkdown(w io.Writer, doc inputDoc) {
	fmt.Fprintf(w, "## Input *%s*\n", doc.name)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "### Overview")
	fmt.Fprintln(w, breakAfterDots(doc.help))
	fmt.Fprintln(w)
	fmt.Fprintln(w, "### Configuration")
	if len(doc.keys) == 0 {
		fmt.Fprintf(w, "No configuration available")
	} else {
		fmt.Fprintf(w, "\nKeys available in the `[input.config]` section:\n\n")
		genConfigKeysMarkdown(w, doc.keys)
	}
}

func genFilterMarkdown(w io.Writer, doc filterDoc) {
	fmt.Fprintf(w, "## Filter *%s*\n", doc.name)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "### Overview")
	fmt.Fprintln(w, breakAfterDots(doc.help))
	fmt.Fprintln(w)
	fmt.Fprintln(w, "### Configuration")
	if len(doc.keys) == 0 {
		fmt.Fprintf(w, "No configuration available")
	} else {
		fmt.Fprintf(w, "\nKeys available in the `[filter.config]` section:\n\n")
		genConfigKeysMarkdown(w, doc.keys)
	}
}

func genOutputMarkdown(w io.Writer, doc outputDoc) {
	const (
		rawString    = "This is a *raw* output, for each record it receives a buffer containing the serialized record, plus a list holding a set of fields (`output.fields` in TOML)."
		nonRawString = "This is a *non-raw* output, it doesn't receive whole records. Instead it receives a list of fields for each record (`output.fields` in TOML)."
	)

	fmt.Fprintf(w, "## Output *%s*\n", doc.name)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "### Overview")
	if doc.raw {
		fmt.Fprintln(w, rawString)
	} else {
		fmt.Fprintln(w, nonRawString)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w)
	fmt.Fprintln(w, breakAfterDots(doc.help))
	fmt.Fprintln(w)
	fmt.Fprintln(w, "### Configuration")
	if len(doc.keys) == 0 {
		fmt.Fprintf(w, "No configuration available")
	} else {
		fmt.Fprintf(w, "\nKeys available in the `[output.config]` section:\n\n")
		genConfigKeysMarkdown(w, doc.keys)
	}
}

func genUploadMarkdown(w io.Writer, doc uploadDoc) {
	fmt.Fprintf(w, "## Upload *%s*\n", doc.name)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "### Overview")
	fmt.Fprintln(w, breakAfterDots(doc.help))
	fmt.Fprintln(w)
	fmt.Fprintln(w, "### Configuration")
	if len(doc.keys) == 0 {
		fmt.Fprintf(w, "No configuration available")
	} else {
		fmt.Fprintf(w, "\nKeys available in the `[upload.config]` section:\n\n")
		genConfigKeysMarkdown(w, doc.keys)
	}
}

func genMetricsMarkdown(w io.Writer, doc metricsDoc) {
	fmt.Fprintf(w, "## Metrics *%s*\n", doc.name)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "### Configuration")
	if len(doc.keys) == 0 {
		fmt.Fprintf(w, "No configuration available")
	} else {
		fmt.Fprintf(w, "\nKeys available in the `[metrics.config]` section:\n\n")
		genConfigKeysMarkdown(w, doc.keys)
	}
}

func genConfigKeysMarkdown(w io.Writer, keys []helpConfigKey) {
	fmt.Fprintln(w, "|Name|Type|Default|Required|Description|")
	fmt.Fprintln(w, "|----|:--:|:-----:|:------:|-----------|")
	for _, k := range keys {
		fmt.Fprintf(w, "| %v| %v| %v| %t| %v|\n", k.name, k.typ, k.def, k.required, k.desc)
	}
	fmt.Fprintln(w)
}
