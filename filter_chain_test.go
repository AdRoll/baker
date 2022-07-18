package baker_test

import (
	"fmt"
	"reflect"
	"strings"
	"sync/atomic"
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
		Inputs: []baker.InputDesc{inputtest.RecordsDesc},
		Filters: append(filter.All,
			newDropOnErrorDefaultTestDesc(true, noerrAction),
			newDropOnErrorDefaultTestDesc(false, noerrAction),
			newDropOnErrorDefaultTestDesc(true, errorAction),
			newDropOnErrorDefaultTestDesc(false, errorAction),
		),
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

func TestDropOnErrorDefault(t *testing.T) {
	t.Run("dropOnError/val=true/default=true/action=error", func(t *testing.T) {
		toml := `
		[[filter]]
		name="dropOnError/default=true/action=error"
		dropOnError = true`

		testFilterChain(t, toml,
			[]string{"foo,bar,baz"},
			nil, // drop
		)
	})
	t.Run("dropOnError/val=true/default=false/action=error", func(t *testing.T) {
		toml := `
		[[filter]]
		name="dropOnError/default=false/action=error"
		dropOnError = true`

		testFilterChain(t, toml,
			[]string{"foo,bar,baz"},
			nil, // drop
		)
	})
	t.Run("dropOnError/val=true/default=true/action=noerror", func(t *testing.T) {
		toml := `
		[[filter]]
		name="dropOnError/default=true/action=noerror"
		dropOnError = true`

		testFilterChain(t, toml,
			[]string{"foo,bar,baz"},
			[]string{"foo,bar,baz"}, // do not drop
		)
	})
	t.Run("dropOnError/val=true/default=false/action=noerror", func(t *testing.T) {
		toml := `
		[[filter]]
		name="dropOnError/default=false/action=noerror"
		dropOnError = true`

		testFilterChain(t, toml,
			[]string{"foo,bar,baz"},
			[]string{"foo,bar,baz"}, // do not drop
		)
	})

	//
	// dropOnError=false
	//

	t.Run("dropOnError/val=false/default=true/action=error", func(t *testing.T) {
		toml := `
		[[filter]]
		name="dropOnError/default=true/action=error"
		dropOnError = false`

		testFilterChain(t, toml,
			[]string{"foo,bar,baz"},
			[]string{"foo,bar,baz"}, // do not drop
		)
	})
	t.Run("dropOnError/val=false/default=false/action=error", func(t *testing.T) {
		toml := `
		[[filter]]
		name="dropOnError/default=false/action=error"
		dropOnError = false`

		testFilterChain(t, toml,
			[]string{"foo,bar,baz"},
			[]string{"foo,bar,baz"}, // do not drop
		)
	})
	t.Run("dropOnError/val=false/default=true/action=noerror", func(t *testing.T) {
		toml := `
		[[filter]]
		name="dropOnError/default=true/action=noerror"
		dropOnError = false`

		testFilterChain(t, toml,
			[]string{"foo,bar,baz"},
			[]string{"foo,bar,baz"}, // do not drop
		)
	})
	t.Run("dropOnError/val=false/default=false/action=noerror", func(t *testing.T) {
		toml := `
		[[filter]]
		name="dropOnError/default=false/action=noerror"
		dropOnError = false`

		testFilterChain(t, toml,
			[]string{"foo,bar,baz"},
			[]string{"foo,bar,baz"}, // do not drop
		)
	})

	//
	// dropOnError=unset
	//

	t.Run("dropOnError/val=unset/default=true/action=error", func(t *testing.T) {
		toml := `
		[[filter]]
		name="dropOnError/default=true/action=error"`

		testFilterChain(t, toml,
			[]string{"foo,bar,baz"},
			nil, // drop
		)
	})
	t.Run("dropOnError/val=unset/default=false/action=error", func(t *testing.T) {
		toml := `
		[[filter]]
		name="dropOnError/default=false/action=error"`

		testFilterChain(t, toml,
			[]string{"foo,bar,baz"},
			[]string{"foo,bar,baz"}, // do not drop
		)
	})
	t.Run("dropOnError/val=unset/default=true/action=noerror", func(t *testing.T) {
		toml := `
		[[filter]]
		name="dropOnError/default=true/action=noerror"`

		testFilterChain(t, toml,
			[]string{"foo,bar,baz"},
			[]string{"foo,bar,baz"}, // do not drop
		)
	})
	t.Run("dropOnError/val=unset/default=false/action=noerror", func(t *testing.T) {
		toml := `
		[[filter]]
		name="dropOnError/default=false/action=noerror"`

		testFilterChain(t, toml,
			[]string{"foo,bar,baz"},
			[]string{"foo,bar,baz"}, // do not drop
		)
	})
}

type filterAction string

const (
	errorAction filterAction = "error"
	noerrAction filterAction = "noerror"
)

// newDropOnErrorDefaultTestDesc generates the description for a filter is used
// to test FilterDesc.DropOnErrorDefault. It either generates errors for all
// records, or for none of them. Also DropOnErrorDefault can be defined in the
// filter description.
func newDropOnErrorDefaultTestDesc(dropOnErrorDefault bool, act filterAction) baker.FilterDesc {
	return baker.FilterDesc{
		Name: fmt.Sprintf("dropOnError/default=%t/action=%s", dropOnErrorDefault, act),
		New: func(baker.FilterParams) (baker.Filter, error) {
			return &testFilter{act: act}, nil
		},
		Config:             &struct{}{},
		DropOnErrorDefault: dropOnErrorDefault,
	}
}

type testFilter struct {
	ndropped int64
	act      filterAction
}

func (f *testFilter) Stats() baker.FilterStats {
	return baker.FilterStats{
		NumFilteredLines: atomic.LoadInt64(&f.ndropped),
	}
}

func (f *testFilter) Process(l baker.Record) error {
	if f.act == errorAction {
		atomic.AddInt64(&f.ndropped, 1)
		return baker.ErrGenericFilterError
	}
	return nil
}
