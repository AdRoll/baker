package upload

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"github.com/AdRoll/baker"
)

var S3Desc = baker.UploadDesc{
	Name:   "S3",
	New:    newS3,
	Config: &S3Config{},
	Help:   "S3Uploader uploads files to a destination on S3 that is relative to SourceBasePath",
}

// S3Config holds the configuration for the S3 uploader.
//
// Each local path sent to the uploader channel in Run is sent to S3.
// The S3 destination of such files is determined by:
// - Bucket: S3 bucket name
// - Prefix: s3://bucket/prefix
// - the final component is the path of the file to upload that is relative
// to SourceBasePath. For example, if SourceBasePath is "/tmp/out",
// and the file to upload is "/tmp/out/foo/bar/file.gz", the final S3 path is:
// s3://bucket/prefix/foo/bar/file.gz.
//
// All files received by the uploader should be absolute and rooted at
// SourceBasePath.
type S3Config struct {
	SourceBasePath string        `help:"Base path used to consider the final S3 path." default:"/tmp/baker/ologs/"`
	Region         string        `help:"S3 region to upload to." default:"us-east-1"`
	Bucket         string        `help:"S3 bucket to upload to. (required)"`
	Prefix         string        `help:"Prefix on the destination bucket" default:"/"`
	StagingPath    string        `help:"Local staging area to copy files to before upload. If empty use a temporary directory" default:"/tmp/baker/ologs/staging/"`
	Retries        int           `help:"Number of retries before a failed upload" default:"3"`
	Concurrency    int           `help:"Number of concurrent workers" default:"5"`
	Interval       time.Duration `help:"Period at which the source path is scanned" default:"15s"`

	// set to a closure that removes the temporary staging directory in case we
	// created it ourselves. noop if the user provided the staging area themselves.
	rmdir func() error
}

func (cfg *S3Config) fillDefaults() error {
	if cfg.Region == "" {
		return errors.New("Region is a required parameter")
	}

	if cfg.Prefix == "" {
		cfg.Prefix = "/"
	}

	if cfg.StagingPath == "" {
		dir, err := ioutil.TempDir("", "baker-s3upload-staging-*")
		if err != nil {
			return fmt.Errorf("can't create staging path: %v", err)
		}
		cfg.StagingPath = dir
		cfg.rmdir = func() error { return os.RemoveAll(dir) }
	} else {
		cfg.rmdir = func() error { return nil } //noop
	}

	if cfg.SourceBasePath == "" {
		return errors.New("SourceBasePath is a required parameter")
	}

	if cfg.Retries < 0 {
		return fmt.Errorf("Retries: invalid number: %v", cfg.Retries)
	}
	if cfg.Retries == 0 {
		cfg.Retries = 3
	}

	if cfg.Concurrency < 0 {
		return fmt.Errorf("Concurrency: invalid number: %v", cfg.Concurrency)
	}
	if cfg.Concurrency == 0 {
		cfg.Concurrency = 5
	}

	if cfg.Interval == 0 {
		cfg.Interval = 15 * time.Second
	}

	return nil
}

type S3 struct {
	Cfg *S3Config

	uploader *s3manager.Uploader
	ticker   *time.Ticker
	wgUpload sync.WaitGroup
	quit     chan struct{}
	stopOnce sync.Once

	totaln   int64
	totalerr int64
	queuedn  int64
}

func newS3(cfg baker.UploadParams) (baker.Upload, error) {
	if cfg.DecodedConfig == nil {
		cfg.DecodedConfig = &S3Config{}
	}
	dcfg := cfg.DecodedConfig.(*S3Config)
	if err := dcfg.fillDefaults(); err != nil {
		return nil, fmt.Errorf("upload.s3: %v", err)
	}

	s3svc := s3.New(session.New(&aws.Config{Region: aws.String(dcfg.Region)}))
	return &S3{
		Cfg:      dcfg,
		uploader: s3manager.NewUploaderWithClient(s3svc),
		quit:     make(chan struct{}),
	}, nil
}

func (u *S3) Run(upch <-chan string) {
	// Start a goroutine in which we periodically look at the source
	// path for files and upload the ones we find.
	u.wgUpload.Add(1)
	go func() {
		ticker := time.NewTicker(u.Cfg.Interval)
		defer func() {
			ticker.Stop()
			if err := u.uploadDirectory(); err != nil {
				log.Error(err)
			}
			u.wgUpload.Done()
		}()

		for {
			select {
			case <-ticker.C:
				if err := u.uploadDirectory(); err != nil {
					log.Error(err)
				}
			case <-u.quit:
				return
			}
		}
	}()

	for sourceFilePath := range upch {
		err := u.move(sourceFilePath)
		atomic.AddInt64(&u.totaln, int64(1))
		atomic.AddInt64(&u.queuedn, int64(1))
		if err != nil {
			log.WithFields(log.Fields{"filepath": sourceFilePath}).WithError(err).Error("Couldn't move")
		}
	}

	// Stop blocks until the upload goroutine has exited.
	u.Stop()
}

func (u *S3) move(sourceFilePath string) error {
	relPath, err := filepath.Rel(u.Cfg.SourceBasePath, sourceFilePath)
	if err != nil {
		return err
	}

	destinationPath := filepath.Join(u.Cfg.StagingPath, relPath)

	dir := path.Dir(destinationPath)
	if err := os.MkdirAll(dir, 0777); err != nil {
		return err
	}

	return os.Rename(sourceFilePath, destinationPath)
}

func (u *S3) Stop() {
	// Stop may be called by the Topology in case of early exit (i.e CTRL-C),
	// in that case u.quit would be closed twice since Stop() is also called
	// by S3Upload.Run(). Both paths are necessary for a graceful exit; to the
	// upload that means making sure all files have been uploaded.
	u.stopOnce.Do(func() {
		// Signal the upload goroutine to not go further the currently
		// initiated call and wait for it to have terminated.
		close(u.quit)
		u.wgUpload.Wait()

		if err := u.Cfg.rmdir(); err != nil {
			log.Errorf("upload.S3: error removing temp folder: %v", err)
		}
	})
}

func (u *S3) Stats() baker.UploadStats {
	bag := make(baker.MetricsBag)
	bag.AddGauge("s3upload.queuedn", float64(atomic.LoadInt64(&u.queuedn)))

	return baker.UploadStats{
		NumProcessedFiles: atomic.LoadInt64(&u.totaln),
		NumErrorFiles:     atomic.LoadInt64(&u.totalerr),
		Metrics:           bag,
	}
}

type sem chan struct{}

func (s sem) incr() { s <- struct{}{} }
func (s sem) decr() { <-s }

func (u *S3) uploadDirectory() error {
	wg := sync.WaitGroup{}

	ctx := log.WithFields(log.Fields{"f": "s3upload.uploadDirectory"})
	ctx.Info("Uploading")
	sem := make(sem, u.Cfg.Concurrency)
	ctx.Info("Starting to walk...")
	err := filepath.Walk(u.Cfg.StagingPath, func(fpath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		ctx.WithFields(log.Fields{"fpath": fpath}).Info("Upload scheduled")
		wg.Add(1)
		sem.incr()
		go func(fpath string) {
			defer func() { sem.decr(); wg.Done() }()

			for i := 0; i < u.Cfg.Retries; i++ {
				if err := s3UploadFile(u.uploader, u.Cfg.Bucket, u.Cfg.Prefix, u.Cfg.StagingPath, fpath); err == nil {
					atomic.AddInt64(&u.totaln, int64(1))
					atomic.AddInt64(&u.queuedn, int64(-1))
					break
				} else {
					atomic.AddInt64(&u.totalerr, int64(1))
					log.WithError(err).WithFields(log.Fields{"retry#": i + 1}).Error("failed upload")
				}
			}
		}(fpath)
		return nil
	})
	ctx.Info("All Scheduling done")
	wg.Wait()

	ctx.Info("All upload done")
	return err
}

func s3UploadFile(uploader *s3manager.Uploader, bucket, prefix, localPath, fpath string) error {
	ctx := log.WithFields(log.Fields{"localPath": localPath, "filepath": fpath})

	rel, err := filepath.Rel(localPath, fpath)
	if err != nil {
		return fmt.Errorf("unable to get relative path: %v", err)
	}

	file, err := os.Open(fpath)
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Error(err)
		}
	}()

	ctx.WithFields(log.Fields{"key": filepath.Join(prefix, rel)}).Info("Uploading")
	result, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: &bucket,
		Key:    aws.String(filepath.Join(prefix, rel)),
		Body:   file,
	})
	if err != nil {
		actualS3Path := fmt.Sprintf("s3://%s/%s/%s", bucket, prefix, rel)
		return fmt.Errorf("error uploading %s to %s: %s", fpath, actualS3Path, err)
	}

	// We should really check that what we uploaded is correct before removing
	if err := os.Remove(fpath); err != nil {
		return err
	}

	ctx.WithField("dst", result.Location).Info("Done")

	return nil
}
