/*
Package baker provides types and functions to build a pipeline for the processing of structured data.

Structured data is represented by the Record interface. LogLine implemets that interface and
represents a csv record.

Using the functions in the package one can build and run a Topology, reading its configuration
from a TOML file.

The package doesn't include any component. They can be found in their respective packages
(baker/input, baker/filter, baker/output and baker/upload).

The README file in the project repository provides additional information and examples:
https://github.com/AdRoll/baker/blob/main/README.md
*/
package baker

import (
	"fmt"
	"time"
)

// Main runs the topology corresponding to the provided configuration.
// Depending on the input, it either blocks forever (daemon) or terminates when
// all the records have been processed (batch).
func Main(cfg *Config, duration time.Duration) error {
	topology, err := NewTopologyFromConfig(cfg)
	if err != nil {
		return fmt.Errorf("can't create topology: %s", err)
	}

	// Start the topology
	topology.Start()

	// Now setup the wait condition for exiting the process.
	// We exit when:
	//   * The topology is finished
	//   * If profiling is active, after profileDuration has elapsed
	topdone := make(chan bool)
	go func() {
		topology.Wait()
		topdone <- true
	}()

	// Begin dump statistics
	stats := NewStatsDumper(topology)
	stopStats := stats.Run()

	var timeout <-chan time.Time
	if duration > 0 {
		timeout = time.After(duration)
	}

	select {
	case <-topdone:
	case <-timeout:
	}

	// Stop the stats dumping goroutine (this also prints stats one last time).
	stopStats()

	return topology.Error()
}
