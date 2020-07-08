package baker_test

import (
	"reflect"
	"testing"

	"github.com/AdRoll/baker/filter"
	"github.com/AdRoll/baker/input"
	"github.com/AdRoll/baker/output"
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

func TestAllInputsHasConfigHelp(t *testing.T) {
	for _, input := range input.AllInputs() {
		assertValidConfigHelp(t, input.Name, input.Config)
	}
}

func TestAllFiltersHasConfigHelp(t *testing.T) {
	for _, filter := range filter.AllFilters() {
		assertValidConfigHelp(t, filter.Name, filter.Config)
	}
}

func TestAllOutputsHasConfigHelp(t *testing.T) {
	for _, output := range output.AllOutputs() {
		assertValidConfigHelp(t, output.Name, output.Config)
	}
}

// TODO: Enable when at least one uploader exists
// func TestAllUploadsHasConfigHelp(t *testing.T) {
// 	for _, upload := range upload.AllUploads() {
// 		assertValidConfigHelp(t, upload.Name, upload.Config)
// 	}
// }
