package baker

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"testing"
)

func TestExampleHelp(t *testing.T) {
	defer os.RemoveAll("./_out")

	cmd := exec.Command("go", "build", "-o", "_out/help", "./examples/help")
	if err := cmd.Run(); err != nil {
		t.Fatalf("error: go build ./examples/help: %v", err)
	}
}

func TestE2EFullTopology(t *testing.T) {
	defer os.RemoveAll("./_out")

	t.Run("basic", testE2EFullTopology(
		"./examples/basic/", "", "", "",
	))

	t.Run("sharding", testE2EFullTopology(
		"./examples/sharding/", "", "", "",
	))

	t.Run("advanced/csv", testE2EFullTopology(
		"./examples/advanced/", "testdata/advanced_csv_example.toml",
		"_out/csv.gz",
		"testdata/advanced_csv.golden",
	))

	t.Run("advanced/csv-0x1e", testE2EFullTopology(
		"./examples/advanced/", "./testdata/advanced_csv_example_0x1e.toml",
		"_out/0x1e.csv.gz",
		"testdata/advanced_csv_0x1e.golden",
	))
}

func testE2EFullTopology(pkg, toml, got, want string) func(t *testing.T) {
	return func(t *testing.T) {
		cmd := exec.Command("go", "run", pkg, toml)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("command failed: %s\n%s", err, string(out))
		}

		if got == "" && want == "" {
			// Only check that the topology builds and runs without errors
			// but do not check output
			return
		}

		ok, err := diff(got, want)
		if err != nil {
			t.Fatalf("diff failed: %s", err)
		}

		if !ok {
			t.Fatalf("files don't match: %q %q", got, want)
		}
	}
}

func diff(f1, f2 string) (bool, error) {
	b1, err := ioutil.ReadFile(f1)
	if err != nil {
		return false, fmt.Errorf("can't read file 1 %q: %s", f1, err)
	}

	b2, err := ioutil.ReadFile(f2)
	if err != nil {
		return false, fmt.Errorf("can't read file 2 %q: %s", f2, err)
	}

	return bytes.Equal(b1, b2), nil
}
