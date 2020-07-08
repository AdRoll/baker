package baker

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/AdRoll/baker/testutil"
)

func TestMerge(t *testing.T) {
	testutil.InitLogger()
	b1 := MetricsBag{}
	b2 := MetricsBag{}

	b1.AddDeltaCounter("delta1", 1)
	b1.AddGauge("gauge1", 2.2)
	b1.AddRawCounter("raw1", 3)

	b1.AddDeltaCounter("delta2", 4)
	b1.AddGauge("gauge2", 5.5)
	b1.AddRawCounter("raw2", 6)

	b2.AddDeltaCounter("delta1", 7)
	b2.AddGauge("gauge1", 8.8)
	b2.AddRawCounter("raw1", 9)

	b2.AddDeltaCounter("delta3", 10)
	b2.AddGauge("gauge3", 11.11)
	b2.AddRawCounter("raw3", 12)

	b1.Merge(b2)

	if b1["d:delta1"] != int64(8) {
		t.Errorf("got %d want 8", b1["d:delta1"])
	}
	if b1["d:delta2"] != int64(4) {
		t.Errorf("got %d want 4", b1["d:delta2"])
	}
	if b1["d:delta3"] != int64(10) {
		t.Errorf("got %d want 10", b1["d:delta3"])
	}

	if b1["g:gauge1"] != float64(5.5) {
		t.Errorf("got %d want 5.5", b1["d:gauge1"])
	}
	if b1["g:gauge2"] != float64(5.5) {
		t.Errorf("got %d want 5.5", b1["d:gauge2"])
	}
	if b1["g:gauge3"] != float64(11.11) {
		t.Errorf("got %d want 11.11", b1["d:gauge3"])
	}

	if b1["c:raw1"] != int64(12) {
		t.Errorf("got %d want 12", b1["d:raw1"])
	}
	if b1["c:raw2"] != int64(6) {
		t.Errorf("got %d want 6", b1["d:raw2"])
	}
	if b1["c:raw3"] != int64(12) {
		t.Errorf("got %d want 12", b1["d:raw3"])
	}
}

func TestMergeHistogram(t *testing.T) {
	testutil.InitLogger()
	t.Run("both non-nil", func(t *testing.T) {
		b1 := MetricsBag{}
		b2 := MetricsBag{}
		b1.AddHistogram("hist", []float64{1, 2})
		b2.AddHistogram("hist", []float64{3, 4})

		b1.Merge(b2)

		if !reflect.DeepEqual(b1["h:hist"], []float64{1, 2, 3, 4}) {
			t.Errorf("got %v want [1, 2, 3, 4]", b1["h:hist"])
		}
	})

	t.Run("b1 nil", func(t *testing.T) {
		b1 := MetricsBag{}
		b2 := MetricsBag{}
		b2.AddHistogram("hist", []float64{3, 4})

		b1.Merge(b2)

		if !reflect.DeepEqual(b1["h:hist"], []float64{3, 4}) {
			t.Errorf("got %v want [3, 4]", b1["h:hist"])
		}
	})

	t.Run("b2 nil", func(t *testing.T) {
		b1 := MetricsBag{}
		b2 := MetricsBag{}
		b1.AddHistogram("hist", []float64{1, 2})

		b1.Merge(b2)

		if !reflect.DeepEqual(b1["h:hist"], []float64{1, 2}) {
			t.Errorf("got %v want [1, 2]", b1["h:hist"])
		}
	})

	t.Run("both nil", func(t *testing.T) {
		b1 := MetricsBag{}
		b2 := MetricsBag{}
		b1.Merge(b2)

		if _, ok := b1["h:hist"]; ok {
			t.Errorf("got %v want nil", b1["h:hist"])
		}
	})
}

func TestMergeTimings(t *testing.T) {
	testutil.InitLogger()
	t.Run("both non-nil", func(t *testing.T) {
		b1 := MetricsBag{}
		b2 := MetricsBag{}
		b1.AddTimings("time", []time.Duration{1, 2})
		b2.AddTimings("time", []time.Duration{3, 4})

		b1.Merge(b2)

		if !reflect.DeepEqual(b1["t:time"], []time.Duration{1, 2, 3, 4}) {
			t.Errorf("got %v want [1, 2, 3, 4]", b1["t:time"])
		}
	})

	t.Run("b1 nil", func(t *testing.T) {
		b1 := MetricsBag{}
		b2 := MetricsBag{}
		b2.AddTimings("time", []time.Duration{3, 4})

		b1.Merge(b2)

		if !reflect.DeepEqual(b1["t:time"], []time.Duration{3, 4}) {
			t.Errorf("got %v want [3, 4]", b1["t:time"])
		}
	})

	t.Run("b2 nil", func(t *testing.T) {
		b1 := MetricsBag{}
		b2 := MetricsBag{}
		b1.AddTimings("time", []time.Duration{1, 2})

		b1.Merge(b2)

		if !reflect.DeepEqual(b1["t:time"], []time.Duration{1, 2}) {
			t.Errorf("got %v want [1, 2]", b1["t:time"])
		}
	})

	t.Run("both nil", func(t *testing.T) {
		b1 := MetricsBag{}
		b2 := MetricsBag{}
		b1.Merge(b2)

		if _, ok := b1["t:time"]; ok {
			t.Errorf("got %v want nil", b1["t:time"])
		}
	})
}

func TestPanicMerge(t *testing.T) {
	testutil.InitLogger()
	defer func() {
		r := recover()
		if r == nil {
			t.Errorf("should have panicked")
		}
		want := `unsupported key "pippo"`
		got := fmt.Sprint(r)
		if got != want {
			t.Errorf("got %v want %s", got, want)
		}
	}()

	b1 := MetricsBag{}
	b2 := MetricsBag{}
	b2["pippo"] = 3

	// The following is the code under test
	b1.Merge(b2)
}
