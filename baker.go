package baker

import (
	"time"

	log "github.com/sirupsen/logrus"
)

func Main(cfg *Config, duration time.Duration) error {
	topology, err := NewTopologyFromConfig(cfg)
	if err != nil {
		// TODO: why log.Fatal? we should return an error here
		log.Fatal(err)
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
