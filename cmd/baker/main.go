package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/AdRoll/baker"
)

var (
	flagHelpConfig = flag.String("help", "", "show help for a `component` (use '*' to dump all)")
	flagVersion    = flag.Bool("version", false, "print version number")
)

var build = "-- unknown --"

func main() {
	// Seed pseudo-random number generation using seconds since the epoch
	rand.Seed(time.Now().Unix())

	flag.Usage = displayProgramUsage
	flag.Parse()

	if *flagHelpConfig != "" {
		baker.PrintHelp(os.Stderr, *flagHelpConfig, components)
		return
	}

	if *flagVersion {
		fmt.Printf("Baker version: %s\n", build)
		return
	}

	if len(flag.Args()) < 1 {
		flag.Usage()
		os.Exit(1)
	}

	f, err := os.Open(flag.Arg(0))
	if err != nil {
		log.Fatal("errors opening config:", err)
	}

	cfg, err := baker.NewConfigFromToml(f, components)
	if err != nil {
		log.Fatal(err)
	}
	f.Close()

	var duration time.Duration
	err = baker.Main(cfg, duration)

	// If there's any fatal error, dump it as Fatal,
	// aborting with a non-zero exit code.
	if err != nil {
		log.Fatal(err)
	}
}
