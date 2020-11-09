package baker

import (
	"flag"
	"fmt"
	"html/template"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

var (
	flagHelpConfig = flag.String("help", "", "show help for a `component` (input/filter/output/upload) (use '*' to dump all)")
	flagVersion    = flag.Bool("version", false, "print build version number")
	flagVerbose    = flag.Bool("v", false, "verbose logging (debug level)")
	flagQuiet      = flag.Bool("q", false, "quiet logging (warn level)")
	flagPretty     = flag.Bool("pretty", false, "human-readable logging (unstructured logging)")
	flagPProf      = flag.String("pprof", "", `run pprof server on host port provided (disabled if ""), use "localhost:"  for a free port`)
)

// Use `-ldflags="-X 'github.com/AdRoll/baker.BuildVersion=someversion'"` when building Baker to set this value
var BuildVersion = "-- unknown --"

// MainCLI starts provides a handy way to quickly create a command-line interface to Baker 
// by providing the list of components available to build and run a topology.
//
// The function includes many utilities that can be configured by command line arguments:
//  -help: Prints available options and components
//  -version: print build version (build with `-ldflags="-X 'github.com/AdRoll/baker.BuildVersion=someversion'"` to set the value)
//  -v: verbose logging (not compatible with -q)
//  -q: quiet logging (not compatible with -v)
//  -pretty: logs in textual format instead of JSON format
//  -pprof: run a pprof server on the provided host:port address
func MainCLI(components Components) error {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stderr)

	// Seed pseudo-random number generation using seconds since the epoch
	rand.Seed(time.Now().Unix())

	// Customise program usage message
	flag.Usage = displayProgramUsage(components)

	flag.Parse()

	if *flagHelpConfig != "" {
		PrintHelp(os.Stderr, *flagHelpConfig, components)
		return nil
	}

	if *flagVersion {
		fmt.Printf("Baker version: %s\n", BuildVersion)
		return nil
	}

	if *flagPProf != "" {
		addr, err := checkHostPort(*flagPProf)
		if err != nil {
			return err
		}
		go func() {
			log.Warnf("running pprof server on %s", addr)
			http.ListenAndServe(addr, nil)
		}()
	}

	if len(flag.Args()) < 1 {
		flag.Usage()
		os.Exit(1)
	}

	if *flagVerbose && *flagQuiet {
		return fmt.Errorf("logging can't both be verbose and quiet!")
	}

	if *flagVerbose {
		log.SetLevel(log.DebugLevel)
	}
	if *flagQuiet {
		log.SetLevel(log.WarnLevel)
	}
	if *flagPretty {
		log.SetFormatter(&log.TextFormatter{})
	}

	f, err := os.Open(flag.Arg(0))
	if err != nil {
		return fmt.Errorf("errors opening config: %v", err)
	}

	cfg, err := NewConfigFromToml(f, components)
	if err != nil {
		return err
	}
	f.Close()

	log.WithField("c", cfg.String()).Info("configuration")

	if err := Main(cfg); err != nil {
		return err
	}

	return nil
}

var programUsageTemplate = template.Must(template.New("Program usage").Parse(`
Baker version: {{ .Build }}

Usage: {{ .ExecName }} [options] TOPOLOGY

TOPOLOGY must be a pathname to a TOML file describing the topology to create.

Options:
{{ .Defaults }}

Available inputs:
{{ range .Components.Inputs }}
  * {{ .Name }}{{ end }}

Available filters:
{{ range .Components.Filters }}
  * {{ .Name }}{{ end }}

Available outputs:
{{ range .Components.Outputs }}
  * {{ .Name }}{{ end }}

Available uploads:
{{ range .Components.Uploads }}
  * {{ .Name }}{{ end }}

`))

func displayProgramUsage(components Components) func() {
	return func() {
		// Structure program usage sections
		type programUsage struct {
			Build      string
			ExecName   string
			Defaults   string
			Components Components
		}

		// Capture command argument defaults
		var defaultsBuilder strings.Builder
		flag.CommandLine.SetOutput(&defaultsBuilder)
		flag.PrintDefaults()

		// Inject program usage data into message template
		if err := programUsageTemplate.Execute(os.Stderr, &programUsage{
			Build:      BuildVersion,
			ExecName:   os.Args[0],
			Defaults:   defaultsBuilder.String(),
			Components: components,
		}); err != nil {
			panic(err)
		}
	}
}

// checkHostPort checks that addr ("host:port" format) is a free tcp port
// suitable for binding a listener.
// NOTE: use 'localhost:' to let the OS find a free "host:port".
func checkHostPort(addr string) (string, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return "", err
	}

	l, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return "", err
	}
	defer l.Close()
	return l.Addr().String(), nil
}
