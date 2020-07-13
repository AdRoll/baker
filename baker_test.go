package baker

import (
	"strings"
	"testing"
)

var dummyInputDesc = InputDesc{
	Name:   "DummyInput",
	New:    newDummyInput,
	Config: &dummyInputConfig{},
}

type dummyInputConfig struct{}
type dummyInput struct{}

func newDummyInput(cfg InputParams) (Input, error) {
	return &dummyInput{}, nil
}

func (d *dummyInput) Run(output chan<- *Data) error {
	return nil
}
func (d *dummyInput) Stats() InputStats {
	return InputStats{}
}
func (d *dummyInput) Stop()              {}
func (d *dummyInput) FreeMem(data *Data) {}

var dummyOutputDesc = OutputDesc{
	Name:   "DummyOut",
	New:    newDummyOutput,
	Config: &dummyOutputConfig{},
}

type dummyOutputConfig struct{}
type dummyOutput struct{}

func newDummyOutput(cfg OutputParams) (Output, error) {
	return &dummyOutput{}, nil
}

func (d *dummyOutput) Run(input <-chan OutputRecord, _ chan<- string) {}

func (d *dummyOutput) Stats() OutputStats { return OutputStats{} }

func (r *dummyOutput) CanShard() bool { return false }

func TestCustomFieldSeparator(t *testing.T) {
	comp := Components{
		Inputs:      []InputDesc{dummyInputDesc},
		Outputs:     []OutputDesc{dummyOutputDesc},
		FieldByName: func(name string) (FieldIndex, bool) { return 0, true },
	}

	t.Run("without separator in conf", func(t *testing.T) {
		toml := `
[input]
name="DummyInput"
[output]
name="DummyOut"
`
		cfg, err := NewConfigFromToml(strings.NewReader(toml), comp)
		if err != nil {
			t.Fatal(err)
		}
		if cfg.CSV.FieldSeparator != "" {
			t.Fatalf("want: %s, got: %s", "", cfg.CSV.FieldSeparator)
		}
		if cfg.fieldSeparator != 44 {
			t.Fatalf("want: %v, got: %v", 44, cfg.fieldSeparator)
		}
	})

	t.Run("with comma separator", func(t *testing.T) {
		toml := `
[csv]
field_separator='2c' # comma
[input]
name="DummyInput"
[output]
name="DummyOut"
		`
		cfg, err := NewConfigFromToml(strings.NewReader(toml), comp)
		if err != nil {
			t.Fatal(err)
		}
		if cfg.CSV.FieldSeparator != "2c" {
			t.Fatalf("want: %s, got: %s", "2c", cfg.CSV.FieldSeparator)
		}
		if cfg.fieldSeparator != 44 {
			t.Fatalf("want: %v, got: %v", 44, cfg.fieldSeparator)
		}
	})

	t.Run("with record separator", func(t *testing.T) {
		toml := `
[csv]
field_separator='1e' # record separator
[input]
name="DummyInput"
[output]
name="DummyOut"
		`
		cfg, err := NewConfigFromToml(strings.NewReader(toml), comp)
		if err != nil {
			t.Fatal(err)
		}
		if cfg.CSV.FieldSeparator != "1e" {
			t.Fatalf("want: %s, got: %s", "1e", cfg.CSV.FieldSeparator)
		}
		if cfg.fieldSeparator != 30 {
			t.Fatalf("want: %v, got: %v", 30, cfg.fieldSeparator)
		}
	})
}
