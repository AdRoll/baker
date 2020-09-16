package baker_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/AdRoll/baker"
	"github.com/AdRoll/baker/input/inputtest"
	"github.com/AdRoll/baker/output/outputtest"
)

func fillComponentsAndLoadConfig(t *testing.T, toml string, user ...baker.UserDesc) (*baker.Config, error) {
	t.Helper()

	const base = `
[input]
name="random"

[output]
name="recorder"
`
	comp := baker.Components{
		Inputs:  []baker.InputDesc{inputtest.RandomDesc},
		Outputs: []baker.OutputDesc{outputtest.RecorderDesc},
		User:    user,
	}

	return baker.NewConfigFromToml(strings.NewReader(base+toml), comp)
}

func TestUserConfigSimple(t *testing.T) {
	// This test checks that a single user configuration is correctly decoded.
	const toml = `
[[user]]
name="MyConfiG"

	[user.config]
	field1 = 1
	field2 = "hello!"`

	type myConfig struct {
		Field1 int
		Field2 string
	}
	mycfg := myConfig{}
	userCfg := baker.UserDesc{Name: "myconfig", Config: &mycfg}

	_, err := fillComponentsAndLoadConfig(t, toml, userCfg)
	if err != nil {
		t.Fatal(err)
	}

	want := myConfig{Field1: 1, Field2: "hello!"}
	if mycfg != want {
		t.Errorf("got %#v, want %#v", mycfg, want)
	}
}

func TestUserConfigMultiple(t *testing.T) {
	// This test checks that we can provide multiple user configurations.
	const toml = `
	# This is user config configA
	[[user]]
	name="configA"

	       [user.config]
	       field1 = 23

	# This is user config configB
	[[user]]
	name="configB"

	       # with a comment
	       [user.config]
	       field1 = ["a", "b", "c", "d"]`

	type configA struct{ Field1 int }
	cfga := configA{}
	ucfga := baker.UserDesc{Name: "configa", Config: &cfga}

	type configB struct{ Field1 []string }
	cfgb := configB{}
	ucfgb := baker.UserDesc{Name: "configb", Config: &cfgb}

	_, err := fillComponentsAndLoadConfig(t, toml, ucfgb, ucfga)
	if err != nil {
		t.Fatal(err)
	}

	wanta := configA{Field1: 23}
	if cfga != wanta {
		t.Errorf("configa: got %#v, want %#v", cfga, wanta)
	}

	wantb := configB{Field1: []string{"a", "b", "c", "d"}}
	if !reflect.DeepEqual(cfgb, wantb) {
		t.Errorf("configb: got %#v, want %#v", cfgb, wantb)
	}
}

func TestUserConfigExtraConfigInTOML(t *testing.T) {
	// This test checks that each user configuration in TOML must be defined
	// of NewConfigFromToml fails.
	const toml = `
	# This is defined in baker.UserDesc
	[[user]]
	name="configA"

	       [user.config]
	       field1 = 23

	# This is not defined in baker.UserDesc
	[[user]]
	name="configB"

	       # with a comment
	       [user.config]
	       field1 = ["a", "b", "c", "d"]`

	type configA struct{ Field1 int }
	cfga := configA{}
	ucfga := baker.UserDesc{Name: "configa", Config: &cfga}

	_, err := fillComponentsAndLoadConfig(t, toml, ucfga)
	if err == nil {
		t.Errorf(`want an error since configB is not defined as a baker.UserDesc, got nil`)
	}
}

func TestUserConfigExtraConfigDefinition(t *testing.T) {
	// This test checks that NewConfigFromToml succeeds if some user
	// configurations have been defined but do not exist in TOML.
	const toml = `
	# This is defined in baker.UserDesc
	[[user]]
	name="configA"

	       [user.config]
	       field1 = 23`

	type configA struct{ Field1 int }
	cfga := configA{}
	ucfga := baker.UserDesc{Name: "configa", Config: &cfga}

	type configB struct{ Field1 []string }
	cfgb := configB{}
	ucfgb := baker.UserDesc{Name: "configb", Config: &cfgb}

	_, err := fillComponentsAndLoadConfig(t, toml, ucfgb, ucfga)
	if err != nil {
		t.Fatal(err)
	}

	wanta := configA{Field1: 23}
	if cfga != wanta {
		t.Errorf("configa: got %#v, want %#v", cfga, wanta)
	}

	wantb := configB{} // zero value
	if !reflect.DeepEqual(cfgb, wantb) {
		t.Errorf("configb: got %#v, want %#v", cfgb, wantb)
	}
}
