package baker_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/AdRoll/baker"
	"github.com/AdRoll/baker/filter/filtertest"
	"github.com/AdRoll/baker/input"
)

func TestRequiredFields(t *testing.T) {
	type (
		test1 struct {
			Name  string
			Value string `help:"field value" required:"false"`
		}

		test2 struct {
			Name  string
			Value string `help:"field value" required:"true"`
		}

		test3 struct {
			Name  string `required:"true"`
			Value string `help:"field value" required:"true"`
		}
	)

	tests := []struct {
		name string
		cfg  interface{}
		want []string
	}{
		{
			name: "no required fields",
			cfg:  &test1{},
			want: nil,
		},
		{
			name: "one required field",
			cfg:  &test2{},
			want: []string{"Value"},
		},
		{
			name: "all required fields",
			cfg:  &test3{},
			want: []string{"Name", "Value"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := baker.RequiredFields(tt.cfg); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RequiredFields() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckRequiredFields(t *testing.T) {
	type (
		test1 struct {
			Name  string
			Value string `help:"field value" required:"false"`
		}

		test2 struct {
			Name  string
			Value string `help:"field value" required:"true"`
		}

		test3 struct {
			Name  string `required:"true"`
			Value string `help:"field value" required:"true"`
		}
	)

	tests := []struct {
		name string
		val  interface{}
		want string
	}{
		{
			name: "no required fields",
			val:  &test1{},
			want: "",
		},
		{
			name: "one missing required field ",
			val:  &test2{Name: "name", Value: ""},
			want: "Value",
		},
		{
			name: "one present required field ",
			val:  &test2{Name: "name", Value: "value"},
			want: "",
		},
		{
			name: "all required fields and all are missing",
			val:  &test3{},
			want: "Name",
		},
		{
			name: "all required fields but the first missing",
			val:  &test3{Value: "value"},
			want: "Name",
		},
		{
			name: "all required fields and all are present",
			val:  &test3{Name: "name", Value: "value"},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := baker.CheckRequiredFields(tt.val); got != tt.want {
				t.Errorf("CheckRequiredFields() = %v, want %v", got, tt.want)
			}
		})
	}
}

func testNewConfigFromTOMLRequiredFields(t *testing.T, name, toml string) {
	t.Run(name, func(t *testing.T) {
		type dummyConfig struct {
			Param1 string
			Param2 string `required:"true"`
		}
		var dummyDesc = baker.OutputDesc{
			Name:   "Dummy",
			New:    func(baker.OutputParams) (baker.Output, error) { return nil, nil },
			Config: &dummyConfig{},
		}

		components := baker.Components{
			Inputs:  []baker.InputDesc{input.ListDesc},
			Filters: []baker.FilterDesc{filtertest.PassThroughDesc},
			Outputs: []baker.OutputDesc{dummyDesc},
		}

		_, err := baker.NewConfigFromToml(strings.NewReader(toml), components)
		if err == nil {
			t.Fatal("expected an error")
		}

		var errReq baker.ErrorRequiredField
		if !errors.As(err, &errReq) {
			t.Fatalf("got %q, want a ErrorRequiredField", err)
		}

		if errReq.Field != "Param2" {
			t.Errorf("got field=%q, want field=%q", errReq.Field, "Param2")
		}
	})
}

func TestNewConfigFromTOMLRequiredField(t *testing.T) {
	toml := `
[input]
name = "List"

[input.config]

[output]
name = "Dummy"
    [output.config]
    param1="this parameter is set"
    #param2="this parameter is not set"
`
	testNewConfigFromTOMLRequiredFields(t, "missing field", toml)

	toml = `
[input]
name = "List"

[output]
name = "Dummy"
`
	testNewConfigFromTOMLRequiredFields(t, "nil config", toml)

	toml = `
	[input]
	name = "List"

	[input.config]

	[output]
	name = "Dummy"
		[output.config]
		PaRam1="this parameter is set"
		#param2="this parameter is not set"
	`
	testNewConfigFromTOMLRequiredFields(t, "case insensitive", toml)
}
