package filter

import (
	"fmt"
	"strings"

	"github.com/AdRoll/baker"
)

const countAndTagHelp = `Publishes a metric that simply counts the number of records passing through, updating a metric of type counter.
In addition, the metric is also tagged with the value of a given, configured, field.
Records having an empty Field value are counted and tagged nonetheless, but they are tagged under the configured tag value: DefaultValue.

#### NOTE
A special attention should be given to the Field used to tag records.
For example, high-cardinality tags might cause performance degradation during tag ingestion and/or visualization and, depending on the metrics client you're using, could incur additional cost.
Something else to keep in mind is that not all values might be valid for the metrics client/system you're using. This filter does not try in any way to validate those.

Finally, it's good to keep in mind that metrics, like other means of observability (logs, tracing, etc.), are provided as best effort and should not influence the program outcome.
As such, it's important to have strong guarantees about the set of possible values for the configured Field, or else it could be necessary to perform some filtering prior to place this filter in your pipeline.
`

var CountAndTagDesc = baker.FilterDesc{
	Name:   "CountAndTag",
	New:    NewCountAndTag,
	Config: &CountAndTagConfig{},
	Help:   countAndTagHelp,
}

type CountAndTagConfig struct {
	Metric       string `help:"Name of the metric of type counter published by this filter" required:"true"`
	Field        string `help:"Field which value is used to to break down the metric by tag values" required:"true"`
	DefaultValue string `help:"Default tag value to use when the value of the configured field is empty" required:"true"`
}

type CountAndTag struct {
	metrics     baker.MetricsClient
	fidx        baker.FieldIndex
	defaultTags []string // special case for default value
	tagPrefix   []byte
	metricName  string
}

func NewCountAndTag(cfg baker.FilterParams) (baker.Filter, error) {
	dcfg := cfg.DecodedConfig.(*CountAndTagConfig)

	fidx, ok := cfg.FieldByName(dcfg.Field)
	if !ok {
		return nil, fmt.Errorf("CountAndTag: unknown field %q", dcfg.Field)
	}

	// Do the most we can now, once, to avoid do it later, many times.
	tagPrefix := strings.ToLower(dcfg.Field) + ":"
	defaultTags := []string{tagPrefix + dcfg.DefaultValue}

	f := &CountAndTag{
		fidx:        fidx,
		metricName:  strings.ToLower(dcfg.Metric),
		metrics:     cfg.Metrics,
		tagPrefix:   []byte(tagPrefix),
		defaultTags: defaultTags,
	}
	return f, nil
}

func (f *CountAndTag) Stats() baker.FilterStats {
	return baker.FilterStats{}
}

func (f *CountAndTag) Process(l baker.Record, next func(baker.Record)) {
	v := l.Get(f.fidx)
	if len(v) == 0 {
		f.metrics.DeltaCountWithTags(f.metricName, 1, f.defaultTags)
	} else {
		tagkv := make([]byte, len(f.tagPrefix)+len(v))
		n := copy(tagkv, f.tagPrefix)
		copy(tagkv[n:], v)
		f.metrics.DeltaCountWithTags(f.metricName, 1, []string{string(tagkv)})
	}
	next(l)
}
