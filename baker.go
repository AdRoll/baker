package baker

import (
	"os"
	"runtime/debug"
	"time"

	log "github.com/sirupsen/logrus"
)

func Main(cfg *Config, duration time.Duration) error {
	/* TODO[opensource]
	   We've seen that the GOGC value depends a lot from the type of job (batch vs always-on)
	   Instead of setting the value based on an environment variable we could add a new general
	   configuration (with default to 800) that could accept both a number or a fixed list of
	   general values like `batch`, `daemon`, etc
	*/
	// This program generates lots of small trash, but the live heap
	// is usually quite small; this means that, by default, the GC
	// triggers quite often. To avoid spending too much CPU time in GC,
	// default to GOGC=800, which trades memory occupation with CPU time.
	if os.Getenv("GOGC") == "" {
		debug.SetGCPercent(800)
	}

	topology, err := NewTopologyFromConfig(cfg)
	if err != nil {
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
