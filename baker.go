/*
Package baker can be used as library to build a baker binary (see the examples/ folder).

In its simplest form (see examples/basic/) a main() function that uses the baker library
must provide a list of components (input, filter, output and upload) as well as a TOML
configuration to the baker functions.

Baker can run as batch, which means that the input component at some point ends (for example
a list of S3 files to process), or as a daemon, a never-ending input (like reading from
a Kinesis stream)

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
