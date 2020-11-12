package baker

import (
	"fmt"
	"io"
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
