package baker_test

import (
	"bytes"
	"flag"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/AdRoll/baker"
	"github.com/AdRoll/baker/filter/filtertest"
	"github.com/AdRoll/baker/input/inputtest"
	"github.com/AdRoll/baker/output/outputtest"
	"github.com/AdRoll/baker/testutil"
	"github.com/AdRoll/baker/upload/uploadtest"
)

type statsInput struct{ inputtest.Base }

func (statsInput) Stats() baker.InputStats {
	bag := baker.MetricsBag{}
	bag.AddDeltaCounter("delta_counter", 1)
	bag.AddGauge("gauge", math.Pi)
	bag.AddHistogram("hist", []float64{1, 2, 3})
	bag.AddTimings("timings", []time.Duration{1 * time.Second, 10 * time.Second, 100 * time.Second})

	return baker.InputStats{
		NumProcessedLines: 93,
		CustomStats:       map[string]string{"k1": "v1", "k2": "v2"},
		Metrics:           bag,
	}
}

type statsFilter struct{ filtertest.Base }

func (statsFilter) Stats() baker.FilterStats {
	bag := baker.MetricsBag{}
	bag.AddDeltaCounter("delta_counter", 3)
	bag.AddGauge("gauge", math.Pi*2)
	bag.AddHistogram("hist", []float64{4, 5, 6, 7})
	bag.AddTimings("timings", []time.Duration{1 * time.Minute, 10 * time.Minute, 100 * time.Minute})

	return baker.FilterStats{
		NumProcessedLines: 67,
		NumFilteredLines:  7,
		Metrics:           bag,
	}
}

type statsOutput struct{ outputtest.Base }

func (statsOutput) Stats() baker.OutputStats {
	bag := baker.MetricsBag{}
	bag.AddDeltaCounter("delta_counter", 7)
	bag.AddGauge("gauge", math.Pi*3)
	bag.AddHistogram("hist", []float64{8, 9, 10, 11})
	bag.AddTimings("timings", []time.Duration{1 * time.Hour, 10 * time.Hour, 100 * time.Hour})

	return baker.OutputStats{
		NumProcessedLines: 53,
		NumErrorLines:     7,
		CustomStats:       map[string]string{"k3": "v3", "k4": "v4"},
		Metrics:           bag,
	}
}

type statsUpload struct{ uploadtest.Base }

func (statsUpload) Stats() baker.UploadStats {
	bag := baker.MetricsBag{}
	bag.AddDeltaCounter("delta_counter", 9)
	bag.AddGauge("gauge", math.Pi*4)
	bag.AddHistogram("hist", []float64{8, 9, 10, 11})
	bag.AddTimings("timings", []time.Duration{1 * time.Microsecond, 10 * time.Microsecond, 100 * time.Microsecond})

	return baker.UploadStats{
		NumProcessedFiles: 17,
		NumErrorFiles:     3,
		CustomStats:       map[string]string{"k5": "v5", "k6": "v6", "k7": "v7"},
		Metrics:           bag,
	}
}

func TestStatsDumper(t *testing.T) {
	buf := &bytes.Buffer{}

	tp := &baker.Topology{
		Input:   &statsInput{},
		Filters: []baker.Filter{&statsFilter{}, &statsFilter{}, &statsFilter{}},
		Output:  []baker.Output{&statsOutput{}, &statsOutput{}},
		Upload:  &statsUpload{},
	}
	sd := baker.NewStatsDumper(tp)
	sd.SetWriter(buf)

	stop := sd.Run()
	// StatsDumper does not print anything the first second
	time.Sleep(1050 * time.Millisecond)
	stop()

	flag.Parse()
	golden := filepath.Join("testdata", t.Name()+".golden")
	if *testutil.UpdateGolden {
		ioutil.WriteFile(golden, buf.Bytes(), os.ModePerm)
		t.Logf("updated: %q", golden)
	}

	testutil.DiffWithGolden(t, buf.Bytes(), golden)
}
