package baker

import (
	"bytes"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"

	log "github.com/sirupsen/logrus"
)

// Topology defines the baker topology, that is how to retrieve records (input),
// how to process them (filter), and where to output the results (output+upload)
type Topology struct {
	Input   Input
	Filters []Filter
	Output  []Output
	Upload  Upload
	Metrics MetricsClient

	inerr     atomic.Value
	inch      chan *Data
	outch     []chan OutputRecord
	rawOutput bool
	upch      chan string

	malformed int64 // count parsing errors and empty records

	mu      sync.RWMutex         // protects invalid map
	invalid map[FieldIndex]int64 // tracks validation errors (by field)

	shard func(l Record) uint64
	chain func(l Record)

	filterProcs int
	outFields   []FieldIndex
	linePool    sync.Pool

	wginp sync.WaitGroup
	wgfil sync.WaitGroup
	wgout sync.WaitGroup
	wgupl sync.WaitGroup

	validate   ValidationFunc
	fieldNames []string // Used by StatsDumper
}

// NewTopologyFromConfig gets a baker configuration and returns a Topology
func NewTopologyFromConfig(cfg *Config) (*Topology, error) {
	var err error

	tp := &Topology{
		filterProcs: cfg.FilterChain.Procs,
		rawOutput:   cfg.Output.desc.Raw,
		validate:    cfg.validate,
		fieldNames:  cfg.fieldNames,
		linePool: sync.Pool{
			New: func() interface{} {
				return cfg.createRecord()
			},
		},
		invalid: make(map[FieldIndex]int64),
	}

	// Create the metrics client first since it's injected into components parameters.
	if cfg.Metrics.Name != "" {
		tp.Metrics, err = cfg.Metrics.desc.New(cfg.Metrics.DecodedConfig)
		if err != nil {
			return nil, fmt.Errorf("error creating metrics interface: %q: %v", cfg.Metrics.Name, err)
		}
	}

	// Assign a dummy client if no one was installed
	if tp.Metrics == nil {
		tp.Metrics = NopMetrics{}
	}

	// * Create input
	inCfg := InputParams{
		ComponentParams{
			DecodedConfig:  cfg.Input.DecodedConfig,
			FieldByName:    cfg.fieldByName,
			FieldNames:     cfg.fieldNames,
			CreateRecord:   cfg.createRecord,
			ValidateRecord: cfg.validate,
			Metrics:        tp.Metrics,
		},
	}
	tp.Input, err = cfg.Input.desc.New(inCfg)
	if err != nil {
		return nil, fmt.Errorf("error creating input: %v", err)
	}

	// * Create filters
	for idx := range cfg.Filter {
		filCfg := FilterParams{
			ComponentParams{
				DecodedConfig:  cfg.Filter[idx].DecodedConfig,
				FieldByName:    cfg.fieldByName,
				FieldNames:     cfg.fieldNames,
				CreateRecord:   cfg.createRecord,
				ValidateRecord: cfg.validate,
				Metrics:        tp.Metrics,
			},
		}
		fil, err := cfg.Filter[idx].desc.New(filCfg)
		if err != nil {
			return nil, fmt.Errorf("error creating filter: %v", err)
		}
		tp.Filters = append(tp.Filters, fil)
	}

	// * Create outputs
	if len(cfg.Output.Fields) == 0 && !tp.rawOutput {
		return nil, fmt.Errorf("error creating output: no \"fields\" specified in [output]")
	}

	for _, fname := range cfg.Output.Fields {
		fidx, ok := cfg.fieldByName(fname)
		if !ok {
			return nil, fmt.Errorf("error creating output: unknown field: %q", fname)
		}
		tp.outFields = append(tp.outFields, fidx)
	}

	for i := 0; i < cfg.Output.Procs; i++ {
		outCfg := OutputParams{
			ComponentParams: ComponentParams{
				DecodedConfig:  cfg.Output.DecodedConfig,
				FieldByName:    cfg.fieldByName,
				FieldNames:     cfg.fieldNames,
				CreateRecord:   cfg.createRecord,
				ValidateRecord: cfg.validate,
				Metrics:        tp.Metrics,
			},
			Index:  i,
			Fields: tp.outFields,
		}
		out, err := cfg.Output.desc.New(outCfg)
		if err != nil {
			return nil, fmt.Errorf("error creating output: %v", err)
		}
		tp.Output = append(tp.Output, out)
	}

	// Create the input-to-filter channel
	tp.inch = make(chan *Data, cfg.Input.ChanSize)

	// Initialize the sharding functions and the output channels.
	// If a sharding function is present, we need one channel per each
	// output worker, and the sharding function will decided where to
	// send each output; if there is no sharding, we create one
	// channel, and the output workers will all fetch from the same.
	tp.outch = make([]chan OutputRecord, cfg.Output.Procs)

	if cfg.Output.Sharding != "" {
		field, ok := cfg.fieldByName(cfg.Output.Sharding)
		if !ok {
			return nil, fmt.Errorf("invalid field: %q", cfg.Output.Sharding)
		}

		tp.shard, ok = cfg.shardingFuncs[field]
		if tp.shard == nil {
			return nil, fmt.Errorf("field not supported for sharding: %q", cfg.Output.Sharding)
		}

		if !tp.Output[0].CanShard() {
			return nil, fmt.Errorf("output component %q does not support sharding", cfg.Output.Name)
		}

		for i := range tp.outch {
			tp.outch[i] = make(chan OutputRecord, cfg.Output.ChanSize)
		}
	} else {
		tp.outch[0] = make(chan OutputRecord, cfg.Output.ChanSize)
	}

	if cfg.Upload.Name != "" {
		upCfg := UploadParams{
			ComponentParams{
				DecodedConfig:  cfg.Upload.DecodedConfig,
				FieldByName:    cfg.fieldByName,
				FieldNames:     cfg.fieldNames,
				CreateRecord:   cfg.createRecord,
				ValidateRecord: cfg.validate,
				Metrics:        tp.Metrics,
			},
		}
		tp.Upload, err = cfg.Upload.desc.New(upCfg)
		if err != nil {
			return nil, fmt.Errorf("error creating upload: %v", err)
		}
	}
	tp.upch = make(chan string)

	// Create the filter chain
	next := tp.filterChainEnd
	for i := len(tp.Filters) - 1; i >= 0; i-- {
		nf := next
		f := tp.Filters[i]
		next = func(l Record) {
			f.Process(l, nf)
		}
	}
	tp.chain = func(l Record) {
		next(l)
		l.Clear()
		tp.linePool.Put(l)
	}

	// Disable validation if required
	if cfg.General.DontValidateFields {
		tp.validate = nil
	}

	return tp, nil
}

// Start starts the Topology, that is start all components.
// This function also intercepts the interrupt signal (ctrl+c)
// starting the graceful shutdown (calling Topology.Stop())
func (t *Topology) Start() {
	// Start the uploader
	t.wgupl.Add(1)
	go func() {
		if t.Upload != nil {
			if err := t.Upload.Run(t.upch); err != nil {
				log.WithError(err).Fatal("Upload returned an error")
			}
		} else {
			// Just consume t.upch if there's no uploader available
			for range t.upch {
				continue
			}
		}
		t.wgupl.Done()
	}()

	// Start a number of output processes equal to the configured
	// [output.procs], each of them reads from its own output channel.
	for idx, out := range t.Output {
		t.wgout.Add(1)
		ch := t.outch[idx]
		if ch == nil {
			ch = t.outch[0]
		}
		go func(out Output) {
			if err := out.Run(ch, t.upch); err != nil {
				log.WithError(err).Fatal("Output returned an error")
			}
			t.wgout.Done()
		}(out)
	}

	// Start the filters
	for i := 0; i < t.filterProcs; i++ {
		t.wgfil.Add(1)
		go func() {
			t.runFilterChain()
			t.wgfil.Done()
		}()
	}

	// Start the input
	t.wginp.Add(1)
	go func() {
		err := t.Input.Run(t.inch)
		if err != nil {
			t.inerr.Store(err)
		}
		t.wginp.Done()
	}()

	stopch := make(chan os.Signal, 1)
	signal.Notify(stopch, os.Interrupt)
	go func() {
		<-stopch
		log.Warn("CTRL+C caught, doing clean shutdown (use CTRL+\\ aka SIGQUIT to abort)")
		t.Stop()
	}()
}

// Stop requires the currently running topology stop safely, but ASAP.
//
// The stop request is forwarded to the input that triggers the chain of stops
// from the components (managed into Topology.Wait).
func (t *Topology) Stop() {
	t.Input.Stop()
}

// Wait until the topology shuts itself down.
//
// This can happen because the input component exits (in a batch topology), or
// in response to a SIGINT signal, that is handled as a clean shutdown request.
func (t *Topology) Wait() {
	t.wginp.Wait()
	close(t.inch)
	t.wgfil.Wait()
	for _, ch := range t.outch {
		if ch != nil {
			close(ch)
		}
	}
	t.wgout.Wait()
	close(t.upch)
	t.wgupl.Wait()
}

// Return the global (sticky) error state of the topology.
//
// Calling this function makes sense after Wait() is complete (before that, it
// is potentially subject to races). Errors from the input components are
// returned here, because they are considered fatals for the topology; all other
// errors (like transient network stuff during output) are not considered fatal,
// and are supposed to be handled within the components themselves.
func (t *Topology) Error() error {
	if err := t.inerr.Load(); err != nil {
		return err.(error)
	}
	return nil
}

func (t *Topology) filterChainEnd(l Record) {
	// Extract fields for output
	var rawOut []byte
	out := make([]string, len(t.outFields))
	if t.rawOutput {
		rawOut = l.ToText(rawOut)
	}
	for idx, f := range t.outFields {
		out[idx] = string(l.Get(f))
	}

	// Calculate sharding
	outch := t.outch[0]
	if t.shard != nil {
		idx := t.shard(l)
		outch = t.outch[int(idx%uint64(len(t.outch)))]
	}
	outch <- OutputRecord{Record: rawOut, Fields: out}
}

func (t *Topology) runFilterChain() {
	mdZero := Metadata{}

	for bakerData := range t.inch {
		data := bakerData.Bytes

		for len(data) > 0 {
			// Split the lines on newlines (without doing memory allocations)
			var line []byte
			if nl := bytes.IndexByte(data, '\n'); nl >= 0 {
				line = data[:nl]
				data = data[nl+1:]
			} else {
				line = data
				data = nil
			}

			// Get a new record from the pool and decode the buffer into it.
			record := t.linePool.Get().(Record)
			err := record.Parse(line, bakerData.Meta)
			if err != nil || len(line) == 0 {
				// Count parse errors or empty records
				atomic.AddInt64(&t.malformed, 1)
				continue
			}

			// Validate against patterns
			if t.validate != nil {
				// call external validation function
				ok, idx := t.validate(record)
				if !ok {
					t.mu.Lock()
					t.invalid[idx]++
					t.mu.Unlock()
					continue
				}
			}

			// Send the logline through the filter chain
			t.chain(record)
		}

		// zero out the common metadata struct.  this doesn't allocate:
		bakerData.Meta = mdZero

		// Give back memory to the input component; it might be able to
		// recycle it, thus avoiding generating too much garbage
		t.Input.FreeMem(bakerData)
	}
}
