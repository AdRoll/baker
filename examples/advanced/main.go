package main

import (
	"flag"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/AdRoll/baker"
)

var flagHelpConfig = flag.String("help", "", "show help for a `component` (use '*' to dump all)")

func main() {
	// Seed pseudo-random number generation using seconds since the epoch
	rand.Seed(time.Now().Unix())

	flag.Usage = displayProgramUsage
	flag.Parse()

	if *flagHelpConfig != "" {
		baker.PrintHelp(os.Stderr, *flagHelpConfig, components)
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
	if err := baker.Main(cfg, duration); err != nil {
		log.Fatal(err)
	}
}
