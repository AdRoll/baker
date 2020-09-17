package input

import (
	"bytes"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/AdRoll/baker"
	"github.com/AdRoll/baker/awsutils"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kinesis"
	log "github.com/sirupsen/logrus"
)

var KTailDesc = baker.InputDesc{
	Name:   "Kinesis",
	New:    NewKTail,
	Config: &KTailConfig{},
	Help: "This input fetches log lines from Kinesis. It listens on a specified stream, and\n" +
		"processes all the shards in that stream. It never exits.\n",
}

type KTailConfig struct {
	AwsRegion string        `help:"AWS region to connect to" default:"us-west-2"`
	Stream    string        `help:"Stream name on Kinesis" required:"true"`
	IdleTime  time.Duration `help:"Time between polls of each shard" default:"100ms"`
}

func (cfg *KTailConfig) fillDefaults() error {
	if cfg.AwsRegion == "" {
		cfg.AwsRegion = "us-west-2"
	}
	var z time.Duration
	if cfg.IdleTime == z {
		cfg.IdleTime = 100 * time.Millisecond
	}

	return nil
}

type KTail struct {
	Cfg  *KTailConfig
	Data chan<- *baker.Data

	stop     int64
	svc      *kinesis.Kinesis
	shards   []*kinesis.Shard
	numLines int64
}

// NewKTail creates a Kinesis tail, and immediately do a first connection to
// get the current shard list.
func NewKTail(cfg baker.InputParams) (baker.Input, error) {
	if cfg.DecodedConfig == nil {
		cfg.DecodedConfig = &KTailConfig{}
	}

	dcfg := cfg.DecodedConfig.(*KTailConfig)
	if err := dcfg.fillDefaults(); err != nil {
		return nil, fmt.Errorf("Kinesis: %s", err)
	}

	sess := session.New(&aws.Config{Region: aws.String(dcfg.AwsRegion)})
	kin := kinesis.New(sess)

	s := &KTail{
		Cfg: dcfg,
		svc: kin,
	}
	if err := s.refreshShards(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *KTail) refreshShards() error {
	ctxLog := log.WithFields(log.Fields{
		"f":    "RefreshShards",
		"name": s.Cfg.Stream,
	})

	ctxLog.Info("refreshing shards")
	params := &kinesis.DescribeStreamInput{
		StreamName: aws.String(s.Cfg.Stream),
	}
	var shards []*kinesis.Shard
	err := s.svc.DescribeStreamPages(params, func(page *kinesis.DescribeStreamOutput, lastPage bool) bool {
		for _, shard := range page.StreamDescription.Shards {
			shards = append(shards, shard)
		}
		return !lastPage
	})
	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		ctxLog.WithFields(log.Fields{"error": err}).Error("failed to init stream")
		return err
	}
	s.shards = shards
	return nil
}

func (s *KTail) ProcessRecords(shard *kinesis.Shard) error {
	ctxLog := log.WithFields(log.Fields{
		"f":      "ProcessRecords",
		"stream": s.Cfg.Stream,
		"shard":  *shard.ShardId,
	})
	params := &kinesis.GetShardIteratorInput{
		ShardId:           aws.String(*shard.ShardId),
		ShardIteratorType: aws.String("LATEST"),
		StreamName:        aws.String(s.Cfg.Stream),
	}
	resp, err := s.svc.GetShardIterator(params)
	if err != nil {
		return err
	}
	nextShardIterator := resp.ShardIterator
	backoff := awsutils.DefaultBackoff
	for atomic.LoadInt64(&s.stop) == 0 {
		// ctxLog.Debug("Iterating")
		start := time.Now()
		params1 := &kinesis.GetRecordsInput{
			// Limit:         aws.Int64(1000),
			ShardIterator: aws.String(*nextShardIterator),
		}
		resp1, err := s.svc.GetRecords(params1)
		if err != nil {
			code := err.(awserr.Error).Code()
			switch code {
			case "ProvisionedThroughputExceededException":
				d := backoff.Duration()
				ctxLog.WithFields(log.Fields{"error": code, "backoff": d}).Error("Reconnecting")
				time.Sleep(d)
				continue

			default:
				ctxLog.WithFields(log.Fields{"error": code}).Error("Unexpected")
				return err
			}
		}
		backoff.Reset()
		if len(resp1.Records) > 0 {
			var nlines int64
			// ctxLog.WithField("records", len(resp1.Records)).Debug("Iterating")
			for _, d := range resp1.Records {
				nlines += int64(bytes.Count(d.Data, []byte{'\n'}))
				s.Data <- &baker.Data{Bytes: d.Data}
			}
			atomic.AddInt64(&s.numLines, nlines)
		} else if resp1.NextShardIterator == nil || err != nil {
			// Technically when NextShareIterator is empty (null in Java) it means that the Shard has been
			// Shut down. Details of this are spread out in the Java KCL.
			// This is the definition of an ended shard
			// https://github.com/awslabs/amazon-kinesis-client/blob/c6e393c13ec348f77b8b08082ba56823776ee48a/src/main/java/com/amazonaws/services/kinesis/clientlibrary/lib/worker/KinesisDataFetcher.java#L59
			// This is how you high-level manage the shutdown of a shard, which basically to just resync them all.
			// https://github.com/awslabs/amazon-kinesis-client/blob/master/src/main/java/com/amazonaws/services/kinesis/clientlibrary/lib/worker/Worker.java#L331
			// This is the inside control loop for consuming shards
			// https://github.com/awslabs/amazon-kinesis-client/blob/master/src/main/java/com/amazonaws/services/kinesis/clientlibrary/lib/worker/ShardConsumer.java#L129
			// This is how you checkpoint according to KCL
			// https://github.com/awslabs/amazon-kinesis-client/blob/c6e393c13ec348f77b8b08082ba56823776ee48a/src/main/java/com/amazonaws/services/kinesis/clientlibrary/lib/worker/RecordProcessorCheckpointer.java#L216
			//
			// result.put(LEASE_KEY_KEY, DynamoUtils.createAttributeValue(lease.getLeaseKey()));
			// result.put(LEASE_COUNTER_KEY, DynamoUtils.createAttributeValue(lease.getLeaseCounter()));
			// if (lease.getLeaseOwner() != null) {
			//     result.put(LEASE_OWNER_KEY, DynamoUtils.createAttributeValue(lease.getLeaseOwner()));
			// }
			//
			// result.put(OWNER_SWITCHES_KEY, DynamoUtils.createAttributeValue(lease.getOwnerSwitchesSinceCheckpoint()));
			// result.put(CHECKPOINT_SEQUENCE_NUMBER_KEY, DynamoUtils.createAttributeValue(lease.getCheckpoint().getSequenceNumber()));
			// result.put(CHECKPOINT_SUBSEQUENCE_NUMBER_KEY, DynamoUtils.createAttributeValue(lease.getCheckpoint().getSubSequenceNumber()));
			// if (lease.getParentShardIds() != null && !lease.getParentShardIds().isEmpty()) {
			//     result.put(PARENT_SHARD_ID_KEY, DynamoUtils.createAttributeValue(lease.getParentShardIds()));
			// }
			break
		}

		nextShardIterator = resp1.NextShardIterator
		total := time.Now().Sub(start)
		time.Sleep(s.Cfg.IdleTime - total)
	}
	return err
}

func (s *KTail) Stop() {
	atomic.StoreInt64(&s.stop, 1)
}

func (s *KTail) Run(data chan<- *baker.Data) error {
	s.Data = data

	var wg sync.WaitGroup
	for _, shard := range s.shards {
		wg.Add(1)
		go func(shard *kinesis.Shard) {
			err := s.ProcessRecords(shard)
			if err != nil {
				fmt.Println(err.Error())
			}
			wg.Done()

		}(shard)
	}

	wg.Wait()
	return nil
}

func (s *KTail) Stats() baker.InputStats {
	return baker.InputStats{
		NumProcessedLines: atomic.LoadInt64(&s.numLines),
	}
}

func (s *KTail) FreeMem(data *baker.Data) {
	// Because of the way the AWS SDK works, we can't reuse
	// the buffer for a furhter call, as each call to GetRecords()
	// will return freshly allocated memory anyway.
	// So nothing to do here
}
