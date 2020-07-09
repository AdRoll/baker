package baker

import (
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/AdRoll/baker/metrics"
	log "github.com/sirupsen/logrus"
)

func countInvalid(invalid *[LogLineNumFields]int64) int64 {
	sum := int64(0)
	for _, i := range invalid {
		sum = sum + i
	}
	return sum
}

// A StatsDumper gathers statistics about all baker components of topology.
type StatsDumper struct {
	t     *Topology
	start time.Time

	lock             sync.Mutex
	prevwlines       int64
	prevrlines       int64
	prevUploads      int64
	prevUploadErrors int64
}

// NewStatsDumper creates and initializes a StatsDumper using t.
func NewStatsDumper(t *Topology) (sd *StatsDumper) {
	return &StatsDumper{t: t}
}

func (sd *StatsDumper) dumpNow() {
	sd.lock.Lock()
	defer sd.lock.Unlock()

	t := sd.t
	nsec := int64(time.Now().UTC().Sub(sd.start).Seconds())
	if nsec == 0 {
		return
	}

	var curwlines int64
	for _, o := range t.Output {
		curwlines += o.Stats().NumProcessedLines
	}
	metrics.RawCount("processed_lines", curwlines)

	istats := t.Input.Stats()
	currlines := istats.NumProcessedLines

	// Collect metrics from input, filters and outputs that we can
	// forward to statsd
	allMetrics := make(MetricsBag)
	allMetrics.Merge(istats.Metrics)

	var filtered int64
	filteredMap := make(map[string]int64)
	for _, f := range t.Filters {
		stats := f.Stats()
		if stats.NumFilteredLines > 0 {
			filtered += stats.NumFilteredLines
			filteredMap[fmt.Sprintf("%T", f)] += filtered
		}
		allMetrics.Merge(stats.Metrics)
	}

	outErrors := int64(0)
	for _, o := range t.Output {
		stats := o.Stats()
		outErrors += stats.NumErrorLines
		allMetrics.Merge(stats.Metrics)
	}

	var numUploadErrors, numUploads int64
	if t.Upload != nil {
		uStats := t.Upload.Stats()
		numUploads = uStats.NumProcessedFiles
		numUploadErrors = uStats.NumErrorFiles
		metrics.RawCount("uploads", numUploads)
		metrics.RawCount("upload_errors", numUploadErrors)
		allMetrics.Merge(uStats.Metrics)
	}

	if numUploads < sd.prevUploads {
		log.Fatalf("numUploads < prevUploads: %d < %d\n", numUploads, sd.prevUploads)
	}

	invalid := countInvalid(&t.invalid)
	parseErrors := t.malformed
	totalErrors := invalid + parseErrors + filtered + outErrors
	metrics.RawCount("error_lines", totalErrors)

	for k, v := range allMetrics {
		switch k[0] {
		case 'c':
			metrics.RawCount(k[2:], v.(int64))
		case 'd':
			metrics.DeltaCount(k[2:], v.(int64))
		case 'g':
			metrics.Gauge(k[2:], v.(float64))
		}
	}

	fmt.Printf("Stats: 1s[w:%d r:%d] total[w:%d r:%d u:%d] speed[w:%d r:%d] errors[p:%d i:%d f:%d o:%d u:%d]\n",
		curwlines-sd.prevwlines, currlines-sd.prevrlines,
		curwlines, currlines, numUploads,
		curwlines/nsec, currlines/nsec,
		parseErrors,
		invalid,
		filtered,
		outErrors,
		numUploadErrors)

	if istats.CustomStats != nil {
		fmt.Printf("--- Input stats: %v\n", istats.CustomStats)
	}

	if invalid > 0 {
		m := make(map[string]int64)
		for f := range t.invalid {
			if t.invalid[f] > 0 {
				name := sd.t.fieldName(FieldIndex(f))
				value := t.invalid[f]
				m[name] = value
				metrics.RawCount("error_lines."+name, int64(value))
			}
		}
		fmt.Printf("--- Validation errors: %v\n", m)
	}

	if filtered > 0 {
		fmt.Printf("--- Filtered lines: %v\n", filteredMap)
	}
	metrics.RawCount("filtered_lines", filtered)

	// Go stats
	metrics.Gauge("runtime.numgoroutines", float64(runtime.NumGoroutine()))

	memstats := runtime.MemStats{}
	runtime.ReadMemStats(&memstats)
	metrics.Gauge("runtime.memstats.mallocs", float64(memstats.Mallocs))
	metrics.Gauge("runtime.memstats.frees", float64(memstats.Frees))
	metrics.Gauge("runtime.memstats.heapalloc", float64(memstats.HeapAlloc))
	metrics.Gauge("runtime.memstats.heapsys", float64(memstats.HeapSys))
	metrics.Gauge("runtime.memstats.heapreleased", float64(memstats.HeapReleased))
	metrics.Gauge("runtime.memstats.heapobjects", float64(memstats.HeapObjects))
	metrics.Gauge("runtime.memstats.stacksys", float64(memstats.StackSys))
	metrics.Gauge("runtime.memstats.numgc", float64(memstats.NumGC))

	sd.prevwlines = curwlines
	sd.prevrlines = currlines
	sd.prevUploads = numUploads
	sd.prevUploadErrors = numUploadErrors
}

// Run starts dumping stats every second on standard output. Call stop() to
// stop periodically dumping stats, this prints stats one last time.
func (sd *StatsDumper) Run() (stop func()) {
	sd.start = time.Now().UTC()

	done := make(chan struct{})
	go func() {
		tick := time.NewTicker(1 * time.Second)
		defer tick.Stop()

		for {
			select {
			case <-done:
				return
			case <-tick.C:
				sd.dumpNow()
			}
		}
	}()

	return func() { close(done); sd.dumpNow() }
}
