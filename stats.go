package baker

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// A StatsDumper gathers statistics about all baker components of topology.
type StatsDumper struct {
	t       *Topology
	start   time.Time
	w       io.Writer     // stats destination
	metrics MetricsClient // metrics implementation to use

	lock             sync.Mutex
	prevwlines       int64
	prevrlines       int64
	prevUploads      int64
	prevUploadErrors int64
}

// NewStatsDumper creates and initializes a StatsDumper using the given
// topology and writing stats on standard output. If also exports metrics
// via the Metrics interface configured with the Topology, if any.
func NewStatsDumper(t *Topology) (sd *StatsDumper) {
	mc := t.metrics
	if t.metrics == nil {
		mc = NopMetrics{}
	}

	return &StatsDumper{t: t, w: os.Stdout, metrics: mc}
}

// SetWriter sets the writer into which stats are written.
// SetWriter must be called before Run().
func (sd *StatsDumper) SetWriter(w io.Writer) { sd.w = w }

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
	sd.metrics.RawCount("processed_lines", curwlines)

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
		sd.metrics.RawCount("uploads", numUploads)
		sd.metrics.RawCount("upload_errors", numUploadErrors)
		allMetrics.Merge(uStats.Metrics)
	}

	if numUploads < sd.prevUploads {
		log.Fatalf("numUploads < prevUploads: %d < %d\n", numUploads, sd.prevUploads)
	}

	invalid := sd.countInvalid()
	parseErrors := t.malformed
	totalErrors := invalid + parseErrors + filtered + outErrors
	sd.metrics.RawCount("error_lines", totalErrors)

	for k, v := range allMetrics {
		switch k[0] {
		case 'c':
			sd.metrics.RawCount(k[2:], v.(int64))
		case 'd':
			sd.metrics.DeltaCount(k[2:], v.(int64))
		case 'g':
			sd.metrics.Gauge(k[2:], v.(float64))
		}
	}

	fmt.Fprintf(sd.w, "Stats: 1s[w:%d r:%d] total[w:%d r:%d u:%d] speed[w:%d r:%d] errors[p:%d i:%d f:%d o:%d u:%d]\n",
		curwlines-sd.prevwlines, currlines-sd.prevrlines,
		curwlines, currlines, numUploads,
		curwlines/nsec, currlines/nsec,
		parseErrors,
		invalid,
		filtered,
		outErrors,
		numUploadErrors)

	if istats.CustomStats != nil {
		fmt.Fprintf(sd.w, "--- Input stats: %v\n", istats.CustomStats)
	}

	if invalid > 0 {
		m := make(map[string]int64)
		for f := range t.invalid {
			if t.invalid[f] > 0 {
				name := sd.t.fieldNames[f]
				value := t.invalid[f]
				m[name] = value
				sd.metrics.RawCount("error_lines."+name, int64(value))
			}
		}
		fmt.Fprintf(sd.w, "--- Validation errors: %v\n", m)
	}

	if filtered > 0 {
		fmt.Fprintf(sd.w, "--- Filtered lines: %v\n", filteredMap)
	}
	sd.metrics.RawCount("filtered_lines", filtered)

	// Go stats
	sd.metrics.Gauge("runtime.numgoroutines", float64(runtime.NumGoroutine()))

	memstats := runtime.MemStats{}
	runtime.ReadMemStats(&memstats)
	sd.metrics.Gauge("runtime.memstats.mallocs", float64(memstats.Mallocs))
	sd.metrics.Gauge("runtime.memstats.frees", float64(memstats.Frees))
	sd.metrics.Gauge("runtime.memstats.heapalloc", float64(memstats.HeapAlloc))
	sd.metrics.Gauge("runtime.memstats.heapsys", float64(memstats.HeapSys))
	sd.metrics.Gauge("runtime.memstats.heapreleased", float64(memstats.HeapReleased))
	sd.metrics.Gauge("runtime.memstats.heapobjects", float64(memstats.HeapObjects))
	sd.metrics.Gauge("runtime.memstats.stacksys", float64(memstats.StackSys))
	sd.metrics.Gauge("runtime.memstats.numgc", float64(memstats.NumGC))

	sd.prevwlines = curwlines
	sd.prevrlines = currlines
	sd.prevUploads = numUploads
	sd.prevUploadErrors = numUploadErrors
}

func (sd *StatsDumper) countInvalid() int64 {
	sd.t.mu.RLock()
	defer sd.t.mu.RUnlock()

	sum := int64(0)
	for _, i := range sd.t.invalid {
		sum = sum + i
	}
	return sum
}

// Run starts dumping stats every second on standard output. Call stop() to
// stop periodically dumping stats, this prints stats one last time.
func (sd *StatsDumper) Run() (stop func()) {
	sd.start = time.Now().UTC()

	quit := make(chan struct{})
	done := make(chan struct{})
	go func() {
		tick := time.NewTicker(1 * time.Second)
		defer tick.Stop()

		for {
			select {
			case <-quit:
				close(done)
				return
			case <-tick.C:
				sd.dumpNow()
			}
		}
	}()

	return func() { close(quit); <-done; sd.dumpNow() }
}
