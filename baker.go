package baker

import (
	"fmt"
	"time"
)

// Main xruns the topology corresponding to the provided configuration.
// Depending on the input, it either blocks forever (daemon) or terminates when
// all the records have been processed (batch).
//
// TODO: duration should probably be removed from here, it's awkward to have to pass 0
// here so as to not make baker terminates early. This is only used for taking profiles
// so I think we should think about another API for that here.
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
