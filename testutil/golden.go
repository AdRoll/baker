package testutil

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"testing"
)

var UpdateGolden = flag.Bool("update", false, "update golden files")

// DiffWithGolden is a test helper that compares the src bytes with a file content whose path
// is provided in the 'golden' argument. If UpdateGolden flag is true, than the golden
// file is updated with the provided new content in 'src'.
func DiffWithGolden(t *testing.T, src []byte, golden string) {
	t.Helper()

	// update golden files if necessary
	if *UpdateGolden {
		if err := ioutil.WriteFile(golden, src, 0644); err != nil {
			t.Errorf("can't update golden file %s: %v", golden, err)
		}
		return
	}

	// get golden
	goldbuf, err := ioutil.ReadFile(golden)
	if err != nil {
		t.Errorf("can't read golden file %s: %v", golden, err)
		return
	}

	DiffBytes(t, golden, "actual", goldbuf, src)
}

// DiffBytes fails the test and shows differences, line by line, if any
func DiffBytes(t *testing.T, aname, bname string, a, b []byte) {
	t.Helper()

	var buf bytes.Buffer // holding long error message

	// compare lengths
	if len(a) != len(b) {
		fmt.Fprintf(&buf, "\ndifferent lengths: len(%s) = %d, len(%s) = %d", aname, len(a), bname, len(b))
	}

	// compare contents
	line := 1
	offs := 0
	for i := 0; i < len(a) && i < len(b); i++ {
		ch := a[i]
		if ch != b[i] {
			fmt.Fprintf(&buf, "\n%s:%d:%d:\n%s", aname, line, i-offs+1, lineAt(a, offs))
			fmt.Fprintf(&buf, "\n%s:%d:%d:\n%s", bname, line, i-offs+1, lineAt(b, offs))
			fmt.Fprintf(&buf, "\n\n")
			break
		}
		if ch == '\n' {
			line++
			offs = i + 1
		}
	}

	if buf.Len() > 0 {
		t.Error(buf.String())
	}
}

// lineAt returns the line in text starting at offset offs.
func lineAt(text []byte, offs int) []byte {
	i := offs
	for i < len(text) && text[i] != '\n' {
		i++
	}
	return text[offs:i]
}
