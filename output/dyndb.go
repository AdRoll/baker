package output

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/juju/ratelimit"
	log "github.com/sirupsen/logrus"

	"github.com/AdRoll/baker"
	"github.com/AdRoll/baker/pkg/awsutils"
)

var DynamoDBDesc = baker.OutputDesc{
	Name:   "DynamoDB",
	New:    NewDynamoDB,
	Config: &DynamoDBConfig{},
	Raw:    false,
	Help: "This output writes the filtered log lines to DynamoDB. It must be\n" +
		"configured specifying the region, the table name, and the columns\n" +
		"to write.\nColumns are specified using the syntax \"t:name\" where \"t\"\n" +
		"is the type of the data, and \"name\" is the name of column. Supported\n" +
		"types are: \"n\" - integers; \"s\" - strings.\n" +
		"The first column (and field) must be the primary key.\n",
}

const nRequests = 25

// Global instances of dynamodb.DynamoDB objects
// We want to reuse the same service across all instances
// of the output because otherwise we trigger a TooManyRequests
// error while accessing the EC2 metadata.
type dynamoGlobals struct {
	lock sync.Mutex
	db   map[string]*dynamodb.DynamoDB
}

var BakerTransport = http.DefaultTransport.(*http.Transport).Clone()

func init() {
	BakerTransport.MaxIdleConns = 1024
	BakerTransport.MaxIdleConnsPerHost = 256
}

func (dg *dynamoGlobals) Get(region string) *dynamodb.DynamoDB {
	dg.lock.Lock()
	defer dg.lock.Unlock()

	if dg.db == nil {
		dg.db = make(map[string]*dynamodb.DynamoDB)
	}
	if db, found := dg.db[region]; found {
		return db
	}
	sess := session.New(&aws.Config{
		Region:                        aws.String(region),
		CredentialsChainVerboseErrors: aws.Bool(true),
		DisableParamValidation:        aws.Bool(true),
		HTTPClient: &http.Client{
			Transport: BakerTransport,
		},
	})
	db := dynamodb.New(sess)
	dg.db[region] = db
	return db
}

var DynamoGlobals dynamoGlobals

// dynamoProcess is a helper to handle parallel writing to multiple
// dynamodb region. Each process encapsulates a single dynamoDB service
// and a goroutine; it provides an asynchronous BeginWriting() to start
// writing to the region in the internal goroutine, and a Wait() method
// to wait until the write has happened.
// This structure is required for high throughput as we don't want to
// start a new goroutine on each write, but always reuse the same one.
type dynamoProcess struct {
	db  *dynamodb.DynamoDB
	in  chan *dynamodb.BatchWriteItemInput
	out chan bool

	region     string
	maxbackoff time.Duration

	Stats struct {
		TotalRetries int64 // Total number of retries that were attempted
	}
}

func newDynamoProcess(db *dynamodb.DynamoDB, region string, maxbackoff time.Duration) *dynamoProcess {
	dp := &dynamoProcess{
		db:         db,
		in:         make(chan *dynamodb.BatchWriteItemInput, 1),
		out:        make(chan bool, 1),
		region:     region,
		maxbackoff: maxbackoff,
	}
	go dp.run()
	return dp
}

func (dp *dynamoProcess) doRequest(inp *dynamodb.BatchWriteItemInput) error {
	var err error
	backoff := awsutils.DefaultBackoff
	deadline := time.Now().Add(dp.maxbackoff)

	for time.Now().Before(deadline) {
		var resp *dynamodb.BatchWriteItemOutput

		req, resp := dp.db.BatchWriteItemRequest(inp)
		err := req.Send()
		if err != nil && !req.IsErrorRetryable() && !req.IsErrorThrottle() {
			// If we got err here, it means that the whole batch failed with a
			// non-retryable error. This must be something "permanent" like
			// wrong table name. Just exit.
			// FIXME: maybe we should differentiate batch mode vs daemon mode,
			// and never exits in daemon mode?
			log.Fatal(err)
			return err
		}

		// If all items were processed, exit
		if len(resp.UnprocessedItems) == 0 {
			return nil
		}

		// There are unprocessed items. This is possibly a transient error (usually
		// throughput error), so schedule a retry after a backoff.
		time.Sleep(backoff.Duration())
		inp = &dynamodb.BatchWriteItemInput{
			RequestItems: resp.UnprocessedItems,
		}
		atomic.AddInt64(&dp.Stats.TotalRetries, 1)
	}

	return err
}

func (dp *dynamoProcess) run() {
	for req := range dp.in {
		if err := dp.doRequest(req); err != nil {
			log.WithError(err).Error("error writing to DynamoDB")
			dp.out <- false
		} else {
			dp.out <- true
		}
	}
}

func (dp *dynamoProcess) BeginWriting(req *dynamodb.BatchWriteItemInput) {
	dp.in <- req
}

func (dp *dynamoProcess) Wait() bool {
	return <-dp.out
}

type DynamoDBConfig struct {
	Regions         []string      `help:"DynamoDB regions to connect to" default:"us-west-2"`
	Table           string        `help:"Name of the table to modify" required:"true"`
	Columns         []string      `help:"Table columns that correspond to each of the fields being written"`
	FlushInterval   time.Duration `help:"Interval at which flush the data to DynamoDB even if we have not reached 25 records" default:"1s"`
	MaxWritesPerSec int64         `help:"Maximum number of writes per second that DynamoDB can accept (0 for unlimited)" default:"0"`
	MaxBackoff      time.Duration `help:"Maximum retry/backoff time in case of errors before giving up" default:"2m"`

	limiter *ratelimit.Bucket
}

func (cfg *DynamoDBConfig) fillDefaults() {
	if cfg.Regions == nil {
		cfg.Regions = []string{"us-west-2"}
	}
	var z time.Duration
	if cfg.FlushInterval == z {
		cfg.FlushInterval = 1 * time.Second
	}
	if cfg.MaxBackoff == z {
		cfg.MaxBackoff = 2 * time.Minute
	}

	if cfg.MaxWritesPerSec > 0 {
		// Create the token-bucket limiter if requested; this is shared across
		// all instances of DyanmoWriter, so we keep it in the config structure
		cfg.limiter = ratelimit.NewBucketWithRate(float64(cfg.MaxWritesPerSec), int64(cfg.MaxWritesPerSec))
	}
}

// DynamoDB is a class to do optimized batched writes to a single DynamoDB table
// with a fixed schema (same number of columns for all records).
type DynamoDB struct {
	// atomically-accessed, keep on top for 64-bit alignment.
	totaln int64 // total processed lines
	errn   int64 // number of lines that were skipped because of errors

	TableName string
	Fields    []baker.FieldIndex
	Columns   []string
	Cfg       *DynamoDBConfig

	lock     sync.Mutex
	dbprocs  []*dynamoProcess
	reqinput *dynamodb.BatchWriteItemInput
	reqbuf   [nRequests]*dynamodb.WriteRequest
	pkeys    [nRequests]string
	timer    *time.Timer
	reqn     int
}

// NewDynamoDB create a new DynamoDB output.
//
// TableName is the name of the DynamoDB table to be written.
// Columns is a slice listing the columns that will be written; the first item in the slice
// *MUST* be the primary key of the table.
func NewDynamoDB(cfg baker.OutputParams) (baker.Output, error) {
	dcfg := cfg.DecodedConfig.(*DynamoDBConfig)
	dcfg.fillDefaults()

	if len(cfg.Fields) == 0 {
		return nil, fmt.Errorf("\"fields\" not specified in [output] configuration")
	}
	if len(cfg.Fields) != len(dcfg.Columns) {
		return nil, fmt.Errorf("\"fields\" and \"columns\" must have the same number of elements")
	}

	for _, c := range dcfg.Columns {
		if len(c) < 3 || c[1] != ':' {
			return nil, fmt.Errorf("invalid format for column (should be type:name): %q", c)
		}
		switch c[0] {
		case 's', 'n':
		default:
			return nil, fmt.Errorf("unsupported column type: %q", c)
		}
	}

	b := &DynamoDB{
		TableName: dcfg.Table,
		Fields:    cfg.Fields,
		Columns:   dcfg.Columns,
		Cfg:       dcfg,
	}

	for _, region := range dcfg.Regions {
		if !awsutils.IsValidRegion(region) {
			return nil, fmt.Errorf("invalid region name: %q", region)
		}
		db := DynamoGlobals.Get(region)
		b.dbprocs = append(b.dbprocs, newDynamoProcess(db, region, dcfg.MaxBackoff))
	}

	// Preallocate request buffers for 25 requests of type Put, with the
	// specified columns
	for idx := range b.reqbuf {
		item := make(map[string]*dynamodb.AttributeValue)
		for _, c := range dcfg.Columns {
			item[c[2:]] = new(dynamodb.AttributeValue)
		}
		b.reqbuf[idx] = &dynamodb.WriteRequest{
			PutRequest: &dynamodb.PutRequest{
				Item: item,
			},
		}
	}
	b.reqinput = &dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]*dynamodb.WriteRequest{
			dcfg.Table: nil,
		},
	}

	b.timer = time.NewTimer(b.Cfg.FlushInterval)
	go func() {
		for range b.timer.C {
			b.Flush()
		}
	}()

	return b, nil
}

func (b *DynamoDB) Flush() {
	b.lock.Lock()
	b.flush()
	b.lock.Unlock()
}

func (b *DynamoDB) NumProcessedRecords() int64 {
	return atomic.LoadInt64(&b.totaln)
}

// Push a new record into DynamoDB. The record is first cached internally, then
// when the batch limit (25) is reached, it is actually written to DyanmoDB.
// So Push() might or might not perform a blocking network request.
// The record is a slice of objects, whose order matches the column orderd that
// was specified when creating the instance in NewDynamoDB().
func (b *DynamoDB) push(record []string) {

	b.lock.Lock()
	defer b.lock.Unlock()

	// Check if this primary key is already present in this batch.
	// This could be a duplicated data in the input stream, but DynamoDb
	// would baffle at a request with duplicated primary keys, so we need
	// to skip it.
	pkey := record[0]
	for i := 0; i < b.reqn; i++ {
		if b.pkeys[i] == pkey {
			log.WithField("key", pkey).Warning("found duplicated primary key")
			return
		}
	}
	b.pkeys[b.reqn] = pkey

	// Fill in the request buffer with the specified record.
	// The AWS SDK exposes the funciton dynamodbattribute.ConvertTo()
	// to convert arbitrary data into a dynamodb.Attribute instance.
	// Unfortunately, that function allocates *tons* of trash, as it forces
	// to allocate the attribute itself, plus one allocation for each field.
	// We prefer to handle this manually to avoid most allocations; note that
	// we also reuse existing instances of dynamodb.Attribute, because we expect
	// all fields to be fixed and to never change between.
	req := b.reqbuf[b.reqn]
	for idx, c := range b.Columns {
		data := record[idx]
		ctype, cname := c[0], c[2:]

		if data == "" {
			delete(req.PutRequest.Item, cname)
			continue
		} else {
			if _, ok := req.PutRequest.Item[cname]; !ok {
				req.PutRequest.Item[cname] = new(dynamodb.AttributeValue)
			}
		}

		switch ctype {
		case 'n':
			req.PutRequest.Item[cname].N = &data
		case 's':
			req.PutRequest.Item[cname].S = &data
		default:
			// should not happen, it's checked in the constructor
			panic(fmt.Errorf("invalid column type"))
		}
	}

	b.reqn++
	if b.reqn == nRequests {
		b.flush()
	} else {
		b.timer.Reset(b.Cfg.FlushInterval)
	}
}

func (b *DynamoDB) flush() {
	if b.reqn == 0 {
		return
	}

	if b.Cfg.limiter != nil {
		// Delay through the rate limiter (if one was requested)
		b.Cfg.limiter.Wait(int64(b.reqn))
	}
	b.reqinput.RequestItems[b.TableName] = b.reqbuf[:b.reqn]

	// Start writing to all regions
	for _, dbproc := range b.dbprocs {
		dbproc.BeginWriting(b.reqinput)
	}

	// Wait for all regions to finish
	for _, dbproc := range b.dbprocs {
		dbproc.Wait()
	}

	atomic.AddInt64(&b.totaln, int64(b.reqn))
	b.reqn = 0
}

func (b *DynamoDB) Run(input <-chan baker.OutputRecord, _ chan<- string) error {
	for lldata := range input {
		b.push(lldata.Fields)
	}
	b.Flush()

	return nil
}

func (b *DynamoDB) Stats() baker.OutputStats {

	bag := make(baker.MetricsBag)
	for _, dbproc := range b.dbprocs {
		name := "dynamodb.retries." + dbproc.region
		value := atomic.LoadInt64(&dbproc.Stats.TotalRetries)
		bag.AddRawCounter(name, value)
	}

	return baker.OutputStats{
		NumProcessedLines: atomic.LoadInt64(&b.totaln),
		NumErrorLines:     atomic.LoadInt64(&b.errn),
		Metrics:           bag,
	}
}

func (b *DynamoDB) CanShard() bool {
	return false
}
