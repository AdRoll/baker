package baker_test

import (
	"io"
	"reflect"
	"testing"

	"github.com/AdRoll/baker"
	"github.com/AdRoll/baker/filter"
	"github.com/AdRoll/baker/input"
	"github.com/AdRoll/baker/output"
	"github.com/AdRoll/baker/upload"
)

func assertValidConfigHelp(t *testing.T, name string, cfg interface{}) {
	t.Helper()

	tCfg := reflect.TypeOf(cfg).Elem()

	if tCfg.Kind() != reflect.Struct {
		t.Errorf("Got %v, struct expected", tCfg.Kind())
	}

	for i := 0; i < tCfg.NumField(); i++ {
		if tCfg.Field(i).PkgPath != "" {
			// This is an unexported field
			continue
		}

		if tCfg.Field(i).Tag.Get("help") == "" {
			t.Errorf("%v is missing the config help for %v", name, tCfg.Field(i).Name)
		}
	}
}

func TestAllInputsHaveConfigHelp(t *testing.T) {
	for _, input := range input.All {
		assertValidConfigHelp(t, input.Name, input.Config)
	}
}

func TestAllFiltersHaveConfigHelp(t *testing.T) {
	for _, filter := range filter.All {
		assertValidConfigHelp(t, filter.Name, filter.Config)
	}
}

func TestAllOutputsHaveConfigHelp(t *testing.T) {
	for _, output := range output.All {
		assertValidConfigHelp(t, output.Name, output.Config)
	}
}

func TestAllUploadsHaveConfigHelp(t *testing.T) {
	for _, upload := range upload.All {
		assertValidConfigHelp(t, upload.Name, upload.Config)
	}
}

func TestPrintHelp(t *testing.T) {
	comp := baker.Components{
		Inputs:  input.All,
		Filters: filter.All,
		Outputs: output.All,
		Uploads: upload.All,
	}
	err := baker.PrintHelp(io.Discard, "*", comp, baker.HelpFormatMarkdown)
	if err != nil {
		t.Fatalf("PrintHelp return err: %v, want nil", err)
	}
}
