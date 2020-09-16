package baker

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"unicode"
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
func PrintHelp(w io.Writer, name string, comp Components) {
	dumpall := name == "*"

	for _, inp := range comp.Inputs {
		if strings.EqualFold(inp.Name, name) || dumpall {
			fmt.Fprintf(w, "=============================================\n")
			fmt.Fprintf(w, "Input: %s\n", inp.Name)
			fmt.Fprintf(w, "=============================================\n")
			fmt.Fprintf(w, inp.Help)
			if hasConfig(inp.Config) {
				fmt.Fprintf(w, "\nKeys available in the [input.config] section:\n\n")
				dumpConfigHelp(w, inp.Config)
			} else {
				fmt.Fprintf(w, "\n(no configuration available)\n\n")
			}
			fmt.Fprintln(w)
			fmt.Fprintln(w)
			if !dumpall {
				return
			}
		}
	}

	for _, fil := range comp.Filters {
		if strings.EqualFold(fil.Name, name) || dumpall {
			fmt.Fprintf(w, "=============================================\n")
			fmt.Fprintf(w, "Filter: %s\n", fil.Name)
			fmt.Fprintf(w, "=============================================\n")
			fmt.Fprintf(w, fil.Help)
			if hasConfig(fil.Config) {
				fmt.Fprintf(w, "\nKeys available in the [filter.config] section:\n\n")
				dumpConfigHelp(w, fil.Config)
			} else {
				fmt.Fprintf(w, "\n(no configuration available)\n\n")
			}
			fmt.Fprintln(w)
			fmt.Fprintln(w)
			if !dumpall {
				return
			}
		}
	}

	for _, out := range comp.Outputs {
		if strings.EqualFold(out.Name, name) || dumpall {
			fmt.Fprintf(w, "=============================================\n")
			fmt.Fprintf(w, "Output: %s\n", out.Name)
			fmt.Fprintf(w, "=============================================\n")
			fmt.Fprintf(w, out.Help)
			if hasConfig(out.Config) {
				fmt.Fprintf(w, "\nKeys available in the [output.config] section:\n\n")
				dumpConfigHelp(w, out.Config)
			} else {
				fmt.Fprintf(w, "\n(no configuration available)\n\n")
			}
			fmt.Fprintln(w)
			fmt.Fprintln(w)
			if !dumpall {
				return
			}
		}
	}

	for _, upl := range comp.Uploads {
		if strings.EqualFold(upl.Name, name) || dumpall {
			fmt.Fprintf(w, "=============================================\n")
			fmt.Fprintf(w, "Upload: %s\n", upl.Name)
			fmt.Fprintf(w, "=============================================\n")
			fmt.Fprintf(w, upl.Help)
			if hasConfig(upl.Config) {
				fmt.Fprintf(w, "\nKeys available in the [upload.config] section:\n\n")
				dumpConfigHelp(w, upl.Config)
			} else {
				fmt.Fprintf(w, "\n(no configuration available)\n\n")
			}
			fmt.Fprintln(w)
			fmt.Fprintln(w)
			if !dumpall {
				return
			}
		}
	}

	if !dumpall {
		fmt.Fprintf(os.Stderr, "Component not found: %s\n", name)
	}
}

func dumpConfigHelp(w io.Writer, cfg interface{}) {
	const sfmt = "%-18s | %-18s | %-26s | "
	const sep = "----------------------------------------------------------------------------------------------------"

	hpad := fmt.Sprintf(sfmt, "", "", "")
	fmt.Fprintf(w, sfmt, "Name", "Type", "Default")
	fmt.Fprintf(w, "Help\n%s\n", sep)

	tf := reflect.TypeOf(cfg).Elem()
	for i := 0; i < tf.NumField(); i++ {
		field := tf.Field(i)

		// skip unexported fields
		if field.PkgPath != "" && !field.Anonymous {
			continue
		}

		var typ string
		switch field.Type.Kind() {
		case reflect.Int:
			typ = "int"
		case reflect.String:
			typ = "string"
		case reflect.Slice:
			switch field.Type.Elem().Kind() {
			case reflect.String:
				typ = "array of strings"
			case reflect.Int:
				typ = "array of ints"
			default:
				panic(field.Type.Elem())
			}
		case reflect.Int64:
			if field.Type.Name() == "Duration" {
				typ = "duration"
			} else {
				typ = "int"
			}
		case reflect.Bool:
			typ = "bool"
		default:
			panic(field.Type.Name())
		}

		help := field.Tag.Get("help")
		def := field.Tag.Get("default")

		fmt.Fprintf(w, sfmt, field.Name, typ, def)
		helpLines := strings.Split(wrapString(help, 40), "\n")
		if len(helpLines) > 0 {
			fmt.Fprint(w, helpLines[0], "\n")
			for _, h := range helpLines[1:] {
				fmt.Fprint(w, hpad, "  ", h, "\n")
			}
		} else {
			fmt.Fprint(w, "\n")
		}
	}

	fmt.Fprint(w, sep, "\n")
}

// wrapString wraps the given string within lim width in characters.
//
// Source: https://github.com/mitchellh/go-wordwrap
// Wrapping is currently naive and only happens at white-space. A future
// version of the library will implement smarter wrapping. This means that
// pathological cases can dramatically reach past the limit, such as a very
// long word.
func wrapString(s string, lim uint) string {
	// Initialize a buffer with a slightly larger size to account for breaks
	init := make([]byte, 0, len(s))
	buf := bytes.NewBuffer(init)

	var current uint
	var wordBuf, spaceBuf bytes.Buffer

	for _, char := range s {
		if char == '\n' {
			if wordBuf.Len() == 0 {
				if current+uint(spaceBuf.Len()) > lim {
					current = 0
				} else {
					current += uint(spaceBuf.Len())
					spaceBuf.WriteTo(buf)
				}
				spaceBuf.Reset()
			} else {
				current += uint(spaceBuf.Len() + wordBuf.Len())
				spaceBuf.WriteTo(buf)
				spaceBuf.Reset()
				wordBuf.WriteTo(buf)
				wordBuf.Reset()
			}
			buf.WriteRune(char)
			current = 0
		} else if unicode.IsSpace(char) {
			if spaceBuf.Len() == 0 || wordBuf.Len() > 0 {
				current += uint(spaceBuf.Len() + wordBuf.Len())
				spaceBuf.WriteTo(buf)
				spaceBuf.Reset()
				wordBuf.WriteTo(buf)
				wordBuf.Reset()
			}

			spaceBuf.WriteRune(char)
		} else {

			wordBuf.WriteRune(char)

			if current+uint(spaceBuf.Len()+wordBuf.Len()) > lim && uint(wordBuf.Len()) < lim {
				buf.WriteRune('\n')
				current = 0
				spaceBuf.Reset()
			}
		}
	}

	if wordBuf.Len() == 0 {
		if current+uint(spaceBuf.Len()) <= lim {
			spaceBuf.WriteTo(buf)
		}
	} else {
		spaceBuf.WriteTo(buf)
		wordBuf.WriteTo(buf)
	}

	return buf.String()
}
