package filter

import (
	"reflect"
	"testing"

	"github.com/AdRoll/baker"
	"github.com/AdRoll/baker/testutil"
)

func TestCountAndTag(t *testing.T) {
	tests := []struct {
		metric      string
		field       string
		defTagValue string
		fieldValues []string // represent the value of 'field' in successive records (as many as len(fieldValues))
		want        []string // list of lines returned by metrics client mock
		wantError   bool
	}{
		{
			metric:      "some_metric",
			field:       "field",
			defTagValue: "",
			fieldValues: []string{"foo", "bar", "baz", "foo"},
			want: []string{
				"delta|name=some_metric|value=1|tag=field:bar",
				"delta|name=some_metric|value=1|tag=field:baz",
				"delta|name=some_metric|value=1|tag=field:foo",
				"delta|name=some_metric|value=1|tag=field:foo",
			},
		},
		{
			metric:      "some_metric",
			field:       "field",
			defTagValue: "",
			fieldValues: []string{"", "", "", "foo"},
			want: []string{
				"delta|name=some_metric|value=1",
				"delta|name=some_metric|value=1",
				"delta|name=some_metric|value=1",
				"delta|name=some_metric|value=1|tag=field:foo",
			},
		},
		{
			metric:      "some_metric",
			field:       "field",
			defTagValue: "<none>",
			fieldValues: []string{"", "", "", "foo"},
			want: []string{
				"delta|name=some_metric|value=1|tag=field:<none>",
				"delta|name=some_metric|value=1|tag=field:<none>",
				"delta|name=some_metric|value=1|tag=field:<none>",
				"delta|name=some_metric|value=1|tag=field:foo",
			},
		},

		// error
		{
			metric:      "some_metric",
			field:       "something", // unknown field
			defTagValue: "<none>",
			fieldValues: []string{""},
			wantError:   true,
		},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			metrics := testutil.MockMetrics{}
			params := baker.FilterParams{
				ComponentParams: baker.ComponentParams{
					FieldByName: func(name string) (baker.FieldIndex, bool) {
						return 0, name == "field" // Only 'field' exists
					},
					Metrics: &metrics,
					DecodedConfig: &CountAndTagConfig{
						Metric:          tt.metric,
						Field:           tt.field,
						DefaultTagValue: tt.defTagValue,
					},
				},
			}

			f, err := NewCountAndTag(params)
			if err != nil {
				if !tt.wantError {
					t.Fatalf("got error: %v", err)
				}
				return
			}

			for _, val := range tt.fieldValues {
				ll := baker.LogLine{}
				if err := ll.Parse([]byte(val), nil); err != nil {
					t.Fatal(err)
				}
				f.Process(&ll, func(baker.Record) {})
			}

			got := metrics.PublishedMetrics("")
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("published metrics,\ngot:\n%+v\n\nwant:\n%v\n", got, tt.want)
			}
		})
	}
}
