package filter

import (
	"fmt"
	"strings"

	"github.com/AdRoll/baker"
)

const countAndTagHelp = `Publish a metric that simply counts all records that pass through, and breaks them down, with metrics tags, by the value of a configured field.
Records having an empty string as Field value are counted nonetheless, using the value configured as DefaultValue as tag value.

NOTE: a special attention should be given in the selection of the Field used to tag records.
For example, high-cardinality tags might cause performance degradation during tag ingestion/visualization, or even incur additional cost.
As such, depending on the origin of the possible Field values, it might be worth filtering the data out.
`

var CountAndTagDesc = baker.FilterDesc{
	Name:   "CountAndTag",
	New:    NewCountAndTag,
	Config: &CountAndTagConfig{},
	Help:   countAndTagHelp,
}

type CountAndTagConfig struct {
	Metric       string `help:"Name of the metric, of type counter, this filter publishes" required:"true"`
	Field        string `help:"Field to read to get tag values" required:"true"`
	DefaultValue string `help:"Default tag value under which records having empty Field are counted" required:"true"`
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
