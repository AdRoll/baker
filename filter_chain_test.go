package baker_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/AdRoll/baker"
	"github.com/AdRoll/baker/filter"
	"github.com/AdRoll/baker/filter_error_handler"
	"github.com/AdRoll/baker/input/inputtest"
	"github.com/AdRoll/baker/output/outputtest"
)

func testFilterChain(tb testing.TB, filterToml string, recordStr, want []string) {
	const src = `
[fields]
names = ["fielda", "fieldb", "fieldc"]

[filterchain]
procs=1

[input]
name="records"

# insert filter(s) here
%s

[output]
name="rawrecorder"
procs=1
fields = ["fielda", "fieldb", "fieldc"]
`
	toml := fmt.Sprintf(src, filterToml)

	components := baker.Components{
		Inputs:              []baker.InputDesc{inputtest.RecordsDesc},
		Filters:             filter.All,
		FilterErrorHandlers: filter_error_handler.All,
		Outputs:             []baker.OutputDesc{outputtest.RawRecorderDesc},
	}

	cfg, err := baker.NewConfigFromToml(strings.NewReader(toml), components)
	if err != nil {
		tb.Fatal(err)
	}

	var records []baker.Record
	for i := range recordStr {
		l := &baker.LogLine{FieldSeparator: ','}
		if err := l.Parse([]byte(recordStr[i]), nil); err != nil {
			tb.Fatal(err)
		}
		records = append(records, l)
	}

	// Setup and feed the input component
	topo, err := baker.NewTopologyFromConfig(cfg)
	if err != nil {
		tb.Fatal(err)
	}

	in := topo.Input.(*inputtest.Records)
	in.Records = records

	topo.Start()
	topo.Wait()
	if err := topo.Error(); err != nil {
		tb.Fatalf("topology error: %v", err)
	}

	out := topo.Output[0].(*outputtest.Recorder)
	var got []string
	for _, rec := range out.Records {
		got = append(got, string(rec.Record))
	}
	if !reflect.DeepEqual(got, want) {
		tb.Fatalf("got = %+v, want %+v", got, want)
	}
}

func TestFilterChain(t *testing.T) {
	t.Run("url-escape", func(t *testing.T) {
		toml := `
		[[filter]]
		name="urlescape"
		
		[filter.config]
		srcfield="fielda"
		dstfield="fieldb"
		`
		testFilterChain(t, toml,
			[]string{"foo,bar,baz"},
			[]string{"foo,foo,baz"},
		)
	})

	t.Run("url-unescape", func(t *testing.T) {
		toml := `
		[[filter]]
		name="urlescape"
		dropOnError=true
		
		[filter.config]
		srcfield="fielda"
		dstfield="fieldb"
		unescape = true

		[[filter.error_handler]]
		name = "ClearFields"
		[filter.error_handler.config]
		fields = ["fieldb"]
		`
		testFilterChain(t, toml, []string{
			"%zzzzz,bar,baz", // fielda is not well-formed, it can't be 'unescaped'
			"%6F,bar,baz",    // fielda is well formed 'url escaped'
		}, []string{
			"%6F,o,baz",
		})
	})

	t.Run("not-null/drop", func(t *testing.T) {
		toml := `
		[[filter]]
		name="notnull"
		dropOnError=true
		
		[filter.config]
		fields= ["fielda"]
		`
		testFilterChain(t, toml, []string{
			",bar,baz", // fielda is empty, the record will be dropped
		}, nil)
	})

	t.Run("not-null/nodrop", func(t *testing.T) {
		toml := `
		[[filter]]
		name="notnull"
		dropOnError=false
		
		[filter.config]
		fields= ["fielda"]
		`
		testFilterChain(t, toml, []string{
			",bar,baz", // fielda is empty, the record will be dropped
		}, []string{",bar,baz"})
	})
}
