package baker

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"testing"
)

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

func testE2EFullTopology(pkg, toml, got, want string) func(t *testing.T) {
	return func(t *testing.T) {
		cmd := exec.Command("go", "run", pkg, toml)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("command failed: %s\n%s", err, string(out))
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

func TestE2EFullTopology(t *testing.T) {
	defer os.RemoveAll("./_out")

	t.Run("advanced/csv-record-sep", testE2EFullTopology(
		"./examples/advanced/", "./testdata/list-clause-files-record-sep.toml",
		"./_out/list-clause-files-record-sep.output.csv.gz",
		"./testdata/list-clause-files-record-sep.golden.csv.gz",
	))

	t.Run("advanced/csv-comma", testE2EFullTopology(
		"./examples/advanced/", "./testdata/list-clause-files-comma-sep.toml",
		"./_out/list-clause-files-comma-sep.output.csv.gz",
		"./testdata/list-clause-files-comma-sep.golden.csv.gz",
	))
}
