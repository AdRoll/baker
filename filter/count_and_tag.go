package filter

import (
	"fmt"
	"strings"

	"github.com/AdRoll/baker"
)

const countAndTagHelp = `Publish a metric that simply counts all records that pass through, using one of the field as value for metric tag.

By default, records having where the chosen field is empty are counted, but not tagged.
You can change that behavior by setting a default tag value for these records, with the DefaultTagValue configuration parameter.

NOTE: a special attention should be given in the selected Field used to tag records.
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
	Metric          string `help:"Name of the metric, of type counter, this filter publishes" required:"true"`
	Field           string `help:"Field to read to get tag values" required:"true"`
	DefaultTagValue string `help:"Default value to use when Field is empty"`
}

type CountAndTag struct {
	metrics    baker.MetricsClient
	fidx       baker.FieldIndex
	fieldName  string
	defValue   string
	metricName string
	tagKey     string
}

func NewCountAndTag(cfg baker.FilterParams) (baker.Filter, error) {
	dcfg := cfg.DecodedConfig.(*CountAndTagConfig)

	fidx, ok := cfg.FieldByName(dcfg.Field)
	if !ok {
		return nil, fmt.Errorf("CountAndTag: unknown field %q", dcfg.Field)
	}

	f := &CountAndTag{
		fidx:       fidx,
		metricName: strings.ToLower(dcfg.Metric),
		fieldName:  strings.ToLower(dcfg.Field),
		tagKey:     strings.ToLower(dcfg.Field),
		metrics:    cfg.Metrics,
		defValue:   dcfg.DefaultTagValue,
	}
	return f, nil
}

func (f *CountAndTag) Stats() baker.FilterStats {
	return baker.FilterStats{}
}

func (f *CountAndTag) Process(l baker.Record, next func(baker.Record)) {
	v := l.Get(f.fidx)
	if len(v) == 0 {
		if f.defValue == "" {
			f.metrics.DeltaCount(f.metricName, 1)
		} else {
			// TODO: pre-concat the default key:value string (even the slice?)
			f.metrics.DeltaCountWithTags(f.metricName, 1, []string{f.fieldName + ":" + f.defValue})
		}
	} else {
		// TODO: measure if necessary to use a strings.Builder here
		f.metrics.DeltaCountWithTags(f.metricName, 1, []string{f.fieldName + ":" + string(v)})
	}
	next(l)
}
