package upload

import (
	"fmt"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"github.com/AdRoll/baker"
	"github.com/AdRoll/baker/input/inputtest"
	"github.com/AdRoll/baker/testutil"
)

var customOutputDesc = baker.OutputDesc{
	Name:   "custom",
	New:    func(baker.OutputParams) (baker.Output, error) { return &customOutput{}, nil },
	Config: &struct{}{},
	Raw:    true,
}

type customOutput struct {
	path  string
	sleep time.Duration // sleep duration after each file written
}

func nthFile(n int) string { return fmt.Sprintf("file.%02d", n) }

func (o *customOutput) Stats() baker.OutputStats { return baker.OutputStats{} }
func (o *customOutput) CanShard() bool           { return false }

func (o *customOutput) Run(in <-chan baker.OutputRecord, upch chan<- string) error {
	// Simulate a new file for each received record
	nfiles := 0
	for range in {
		fpath := path.Join(o.path, nthFile(nfiles))
		if err := os.WriteFile(fpath, nil, 0664); err != nil {
			return err
		}
		nfiles++
		upch <- fpath
		time.Sleep(o.sleep)
	}

	return nil
}

func testIntegrationS3(callStop bool) func(t *testing.T) {
	return func(t *testing.T) {
		defer testutil.DisableLogging()()

		toml := `
[fields]
names=["f0", "f1", "f2", "f3"]

[input]
name="records"

[output]
name="custom"
procs=1

[upload]
name="s3"

	[upload.config]
	sourcebasepath=%q
	stagingpath=%q
	bucket="my-bucket"
	interval="10ms"
	retries=1
	concurrency=1`

		/* Configure the pipeline */

		comp := baker.Components{
			Inputs:  []baker.InputDesc{inputtest.RecordsDesc},
			Outputs: []baker.OutputDesc{customOutputDesc},
			Uploads: []baker.UploadDesc{S3Desc},
		}

		basePath, stagingPath := t.TempDir(), t.TempDir()
		r := strings.NewReader(fmt.Sprintf(toml, basePath, stagingPath))
		cfg, err := baker.NewConfigFromToml(r, comp)
		if err != nil {
			t.Fatal(err)
		}

		topo, err := baker.NewTopologyFromConfig(cfg)
		if err != nil {
			t.Fatal(err)
		}

		// Upload: mock AWS S3
		u := topo.Upload.(*S3)
		s, ops, params := mockS3Service(false)
		u.uploader = s3manager.NewUploaderWithClient(s)

		// Output: provide the path in which files should be written
		o := topo.Output[0].(*customOutput)
		o.path = basePath

		const nfiles = 100

		// Input: create some dummy records
		// since the custom output is creating one file per received record,
		// we create 'nfiles' records.
		var records []baker.Record
		for i := 0; i < nfiles; i++ {
			l := baker.LogLine{}
			l.Parse([]byte("dummyrecord"), nil)
			records = append(records, &l)
		}
		topo.Input.(*inputtest.Records).Records = records

		/* Run the pipeline */

		topo.Start()
		if callStop {
			time.Sleep(100 * time.Millisecond)
			topo.Stop()
		}
		topo.Wait()
		if err := topo.Error(); err != nil {
			t.Fatalf("topology error: %v", err)
		}

		/* Checks */

		// Check files have been all uploaded
		for i := range *ops {
			if (*ops)[i] != "PutObject" {
				t.Errorf("got operation %d/%d = %q, want %q", i, nfiles, (*ops)[i], "PutObject")
			}
		}
		// *s3.PutObjectInput
		for i := range *params {
			put := (*params)[i].(*s3.PutObjectInput)
			if *put.Bucket != "my-bucket" {
				t.Errorf("got operation %d/%d bucket = %q, want %q", i, nfiles, *put.Bucket, "my-bucket")
			}

			wantKey := "/" + nthFile(i)
			if *put.Key != wantKey {
				t.Errorf("got operation %d/%d key = %q, want %q", i, nfiles, *put.Key, wantKey)
			}
		}
	}
}

func TestIntegrationS3(t *testing.T) {
	// This test configures a pipelines from the ground up by passing it a TOML,
	// mocks S3 to not rely on an external service. The created topology is then
	// ran in conditions that are very similar to what would happen in production
	// with a batch workload (i.e process an pre-specified amount of record and
	// exits).
	//
	// The topology should end by itself without any error, we also checks that
	t.Run("topology.wait", testIntegrationS3(false))

	// the expected number of files have been uploaded to their expected bucket/key.
	// On the second subtest we immeditately stop the topology after starting it,
	// but the outcome should be the same since we wait enough time for the input
	// to produce the records, which means topology.Stop() should block until:
	//  - all records have been processed,
	//  - all files have been written by the output
	//  - all files have been uplodaed by the S3 upload
	t.Run("topology.stop", testIntegrationS3(true))
}
