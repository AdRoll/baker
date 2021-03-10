package input

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"regexp"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/vmware/vmware-go-kcl/clientlibrary/config"
	"github.com/vmware/vmware-go-kcl/clientlibrary/interfaces"
	"github.com/vmware/vmware-go-kcl/clientlibrary/worker"
	"github.com/vmware/vmware-go-kcl/logger"

	"github.com/AdRoll/baker"
)

// KCLDesc describes the KCL input.
var KCLDesc = baker.InputDesc{
	Name:   "KCL",
	New:    NewKCL,
	Config: &KCLConfig{},
	Help: "This input fetches records from Kinesis with KCL. It consumes a specified stream, and\n" +
		"processes all shards in that stream. It never exits.\n" +
		"Multiple baker instances can consume the same stream, in that case the KCL will take care of\n" +
		"balancing the shards between workers. Careful (shard stealing is not implemented yet).\n" +
		"Resharding on the producer side is automatically handled by the KCL that will distribute\n" +
		"the shards among KCL workers.",
}

// KCLConfig is the configuration for the KCL input.
type KCLConfig struct {
	AwsRegion       string        `help:"AWS region to connect to" default:"us-west-2"`
	Stream          string        `help:"Name of Kinesis stream" required:"true"`
	AppName         string        `help:"Used by KCL to allow multiple app to consume the same stream." required:"true"`
	MaxShards       int           `help:"Max shards this Worker can handle at a time" default:"32767"`
	ShardSync       time.Duration `help:"Time between tasks to sync leases and Kinesis shards" default:"60s"`
	InitialPosition string        `help:"Position in the stream where a new application should start from. Values: LATEST or TRIM_HORIZON" default:"LATEST"`
	initialPosition config.InitialPositionInStream
}

var appNameRx = regexp.MustCompile(`^[a-zA-Z_0-9]+$`)

func (cfg *KCLConfig) validate() error {
	if !appNameRx.MatchString(cfg.AppName) {
		return fmt.Errorf("invalid 'AppName' '%s', accepts only [A-Za-z0-9_]+", cfg.AppName)
	}
	if cfg.InitialPosition != "LATEST" && cfg.InitialPosition != "TRIM_HORIZON" {
		return fmt.Errorf("invalid 'InitialPosition' '%s', accepts only 'LATEST' or 'TRIM_HORIZON'", cfg.InitialPosition)
	}
	return nil
}

func (cfg *KCLConfig) fillDefaults() {
	if cfg.AwsRegion == "" {
		cfg.AwsRegion = "us-west-2"
	}
	switch cfg.InitialPosition {
	case "", "LATEST":
		cfg.initialPosition = config.LATEST
	case "TRIM_HORIZON":
		cfg.initialPosition = config.TRIM_HORIZON
	}
	if cfg.MaxShards == 0 {
		cfg.MaxShards = math.MaxInt16
	}
	if cfg.ShardSync == 0 {
		cfg.ShardSync = time.Minute
	}
}

// KCL is a Baker input reading from Kinesis with the KCL (Kinesis Client Library).
type KCL struct {
	// atomically accessed (leave on top of the struct)
	nlines, nshards int64         // counters
	done            chan struct{} // signal shutdown request from baker to the KCL worker
	inch            chan<- *baker.Data
	streamShards    int // Number of shards (ACTIVE+CLOSED) in the KCL stream
	cfg             *KCLConfig

	workerCfg *config.KinesisClientLibConfiguration
	metrics   kclDatadogMetrics
}

// generateWorkerID generates an unique ID for currrent worker, based off hostname and an UUID.
func generateWorkerID() (string, error) {
	host, err := os.Hostname()
	if err != nil {
		return "", fmt.Errorf("can't generate workerID: %v", err)
	}
	uid, err := uuid.NewUUID()
	if err != nil {
		return "", fmt.Errorf("can't generate workerID: %v", err)
	}
	return fmt.Sprintf("%s-%s", host, uid), nil
}

// NewKCL creates a new KCL.
func NewKCL(cfg baker.InputParams) (baker.Input, error) {
	// Read and validate KCL configuration
	dcfg := cfg.DecodedConfig.(*KCLConfig)
	dcfg.fillDefaults()
	if err := dcfg.validate(); err != nil {
		return nil, fmt.Errorf("can't create KCL input: %v", err)
	}

	// Generate constants variables
	workerID, err := generateWorkerID()
	if err != nil {
		return nil, fmt.Errorf("can't create KCL input: %v", err)
	}

	const (
		// Leases not renewed within this period will be claimed by others
		leaseDuration = 60 * time.Second
		// Period before the end of lease during which a lease is refreshed by the owner
		leaseRefreshPeriod = 20 * time.Second
		// Max records to read per Kinesis getRecords() call
		maxRecords = 10000
	)

	kcl := &KCL{
		cfg:     dcfg,
		done:    make(chan struct{}),
		metrics: kclDatadogMetrics{metricsClient: cfg.Metrics},
	}
	kcl.workerCfg = config.NewKinesisClientLibConfig(dcfg.AppName, dcfg.Stream, dcfg.AwsRegion, workerID).
		WithMaxRecords(maxRecords).
		WithMaxLeasesForWorker(dcfg.MaxShards).
		WithShardSyncIntervalMillis(int(dcfg.ShardSync / time.Millisecond)).
		WithFailoverTimeMillis(int(leaseDuration / time.Millisecond)).
		WithLeaseRefreshPeriodMillis(int(leaseRefreshPeriod / time.Millisecond)).
		WithInitialPositionInStream(dcfg.initialPosition).
		WithMonitoringService(&kcl.metrics).
		WithLogger(logger.NewLogrusLogger(log.StandardLogger()))

	streamShards, err := kcl.totalShards()
	if err != nil {
		return kcl, err
	}
	kcl.streamShards = streamShards
	log.Infof("Total shards for the stream: %d", streamShards)

	kcl.metrics.nshards = &kcl.nshards
	return kcl, nil
}

func (k *KCL) totalShards() (int, error) {
	s, err := session.NewSession(&aws.Config{
		Region:      aws.String(k.cfg.AwsRegion),
		Endpoint:    aws.String(k.workerCfg.KinesisEndpoint),
		Credentials: k.workerCfg.KinesisCredentials,
	})

	if err != nil {
		return 0, err
	}

	kc := kinesis.New(s)

	var totalShards int
	args := &kinesis.ListShardsInput{StreamName: aws.String(k.cfg.Stream)}
	for {
		resp, err := kc.ListShards(args)
		if err != nil {
			log.Errorf("Error in ListShards: %s Error: %+v Request: %s", k.cfg.Stream, err, args)
			return 0, err
		}

		totalShards += len(resp.Shards)

		if resp.NextToken == nil {
			break
		}
		// The use of NextToken requires StreamName to be absent
		args = &kinesis.ListShardsInput{NextToken: resp.NextToken}
	}

	return totalShards, nil
}

// Stop implements baker.Input
func (k *KCL) Stop() {
	close(k.done)
}

// Run implements baker.Input.
func (k *KCL) Run(inch chan<- *baker.Data) error {
	k.inch = inch

	wk := worker.NewWorker(k, k.workerCfg)
	if err := wk.Start(); err != nil {
		return fmt.Errorf("input: kcl: can't start the worker with: %v", err)
	}
	defer wk.Shutdown()

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	// Run a loop that periodically checks the number of shards available in the stream,
	// quitting baker as soon as the number changes
	for {
		select {
		case <-k.done:
			return nil
		case <-ticker.C:
			log.Debug("Refreshing shards number")
			n, err := k.totalShards()
			if err != nil {
				log.Errorf("Error refreshing the shards number: %v", err)
				continue
			}
			log.Debugf("Total shards: %d, new shards count: %d", k.streamShards, n)
			if k.streamShards != n {
				log.Info("Shard number has changed, shutting down")
				return nil
			}
		}
	}
}

// Stats implements baker.Input
func (k *KCL) Stats() baker.InputStats {
	bag := make(baker.MetricsBag)
	bag.AddGauge("kcl.shards", float64(atomic.LoadInt64(&k.nshards)))

	return baker.InputStats{
		NumProcessedLines: atomic.LoadInt64(&k.nlines),
		Metrics:           bag,
	}
}

// FreeMem implements baker.Input
func (k *KCL) FreeMem(data *baker.Data) {
	// Because of the way the AWS SDK works, we can't reuse
	// the buffer for a further call, as each call to GetRecords()
	// will return freshly allocated memory anyway.
	// So nothing to do here
}

// CreateProcessor implements interfaces.IRecordProcessorFactory.
func (k *KCL) CreateProcessor() interfaces.IRecordProcessor {
	return &recordProcessor{
		inch:    k.inch,
		metrics: &k.metrics,
		nlines:  &k.nlines,
	}
}

// recordProcessor implements KCL IRecordProcessor.
type recordProcessor struct {
	nlines *int64 // per-worker number of processed lines (exposed via baker stats)

	inch chan<- *baker.Data

	metrics *kclDatadogMetrics
	shardID string   // ID of the shard this processor consumes
	tags    []string // tags for metrics to associate with this record processor
}

// Shutdown is invoked by the Amazon Kinesis Client Library to indicate it will
// no longer send data records to this RecordProcessor instance.
func (p *recordProcessor) Shutdown(input *interfaces.ShutdownInput) {
	log.WithFields(log.Fields{
		"shard":  p.shardID,
		"reason": aws.StringValue(interfaces.ShutdownReasonMessage(input.ShutdownReason)),
	}).Info("Shutting down a KCL record processor")

	// The shard is closed and completely read, so we checkpoint the nil value that informs
	// vmware-go-kcl about that fact
	if input.ShutdownReason == interfaces.TERMINATE {
		if err := input.Checkpointer.Checkpoint(nil); err != nil {
			log.Errorf("Error checkpointing nil: %v", err)
		}
	}
}

// Initialize is invoked by the Amazon Kinesis Client Library before data
// records are delivered to the RecordProcessor instance (via processRecords).
func (p *recordProcessor) Initialize(input *interfaces.InitializationInput) {
	p.shardID = input.ShardId
	p.tags = []string{fmt.Sprintf("shard:%s", p.shardID)}
	log.WithFields(log.Fields{
		"shard":      input.ShardId,
		"checkpoint": aws.StringValue(input.ExtendedSequenceNumber.SequenceNumber)}).
		Info("Initializing a new RecordProcessor")
}

// ProcessRecords process data records. vmware kcl will invoke this method to
// deliver data records.
//
// Upon fail over, the new instance will get records with sequence number greater
// than checkpoint position for each partition key.
//
// 'input' provides the records to be processed as well as information and
// capabilities related to them (eg checkpointing).
func (p *recordProcessor) ProcessRecords(input *interfaces.ProcessRecordsInput) {
	// Control read throughput to prevent throttling.
	//
	// Kinesis imposes limits on GetRecords, see
	// https://docs.aws.amazon.com/streams/latest/dev/service-sizes-and-limits.html
	//
	// Each shard can support up to a maximum total data read rate of 2 MiB
	// per second via GetRecords. If a call to GetRecords returns 10 MiB, the
	// maximum size GetRecords is allowed to return, subsequent calls made
	// within the next 5 seconds will meet a ProvisionedThroughputExceededException.
	//
	// Limiting the number of records per call would work but would increase
	// the number of performed IO syscalls and will increase the risk to meet
	// the limits imposed by AWS on API calls.
	//
	// The strategy we're using is to not limit MaxRecords but sleeping for 6s.
	// Doing so, we're guaranteed to never exceed the per-shard read througput
	// limit of 2MB/s, while being close to it on data peaks. This has the
	// added advantage of reducing the number of IO syscalls.
	time.Sleep(6 * time.Second)

	// Skip if no records
	if len(input.Records) == 0 {
		log.Debug("No records to process")
		return
	}

	// Send the records to Baker pipeline
	var nlines int64
	for _, v := range input.Records {
		nlines += int64(bytes.Count(v.Data, []byte{'\n'}))
		p.inch <- &baker.Data{Bytes: v.Data}
	}

	// Increment the total number of lines processed by the KCL worker.
	// note: p.nlines is shared among all record processors
	atomic.AddInt64(p.nlines, nlines)

	// Checkpoint it after processing this batch
	lastRecordSequenceNumber := input.Records[len(input.Records)-1].SequenceNumber
	log.Debugf("Processed %d records: checkpoint=%s, msBehindLatest=%v", len(input.Records), aws.StringValue(lastRecordSequenceNumber), input.MillisBehindLatest)
	if err := input.Checkpointer.Checkpoint(lastRecordSequenceNumber); err != nil {
		log.Errorf("Error checkpointing at %s", *lastRecordSequenceNumber)
	}
}

// kclMetrics implements kcl metrics.MonitoringService.
type kclDatadogMetrics struct {
	nshards       *int64 // keep track of the current number of shards (exposed via baker stats)
	metricsClient baker.MetricsClient
}

func (m *kclDatadogMetrics) Init(appname, streamname, workerID string) error { return nil }
func (m *kclDatadogMetrics) Start() error                                    { return nil }
func (m *kclDatadogMetrics) Shutdown()                                       {}

func (m *kclDatadogMetrics) LeaseGained(shard string) { atomic.AddInt64(m.nshards, 1) }
func (m *kclDatadogMetrics) LeaseLost(shard string)   { atomic.AddInt64(m.nshards, -1) }

func (m *kclDatadogMetrics) LeaseRenewed(shard string) {
	m.metricsClient.DeltaCountWithTags("leases_renewals", 1, []string{"shard:" + shard})
}

func (m *kclDatadogMetrics) IncrRecordsProcessed(shard string, count int) {
	m.metricsClient.DeltaCountWithTags("processed_records", int64(count), []string{"shard:" + shard})
}

func (m *kclDatadogMetrics) IncrBytesProcessed(shard string, count int64) {
	m.metricsClient.DeltaCountWithTags("processed_bytes", count, []string{"shard:" + shard})
}

func (m *kclDatadogMetrics) MillisBehindLatest(shard string, ms float64) {
	m.metricsClient.GaugeWithTags("ms_behind_latest_milliseconds", ms, []string{"shard:" + shard})
}

func (m *kclDatadogMetrics) RecordGetRecordsTime(shard string, ms float64) {
	t := time.Duration(ms) * time.Millisecond
	m.metricsClient.DurationWithTags("get_records_time_milliseconds", t, []string{"shard:" + shard})
}

func (m *kclDatadogMetrics) RecordProcessRecordsTime(shard string, ms float64) {
	t := time.Duration(ms) * time.Millisecond
	m.metricsClient.DurationWithTags("process_records_time_milliseconds", t, []string{"shard:" + shard})
}
