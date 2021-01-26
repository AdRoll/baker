// +build linux darwin

package inpututils

import (
	"bytes"
	"errors"
	"io"
	"os/exec"
	"strings"
	"syscall"
)

// fastReader is an API-compatible drop-in replacement
// for a compressed stream reader, That achieves a higher
// decoding speed by spawning an external decompressing process
// instance and pipeing data through it.
// For example Go's native gzip implementation is about 2x slower at
// decompressing data compared to zlib (mostly due to Go compiler
// inefficiencies). So for tasks where the gzip decoding
// speed is important, this is a quick workaround that doesn't
// require cgo.
// zcat is part of the gzip package and comes preinstalled on
// most Linux distributions and on OSX.
// zstdcat can easily be installed on osx and linux
type fastReader struct {
	io.ReadCloser
	command []string
	stderr  bytes.Buffer
	close   func() error
}

func newFastGzReader(r io.Reader) (*fastReader, error) {
	return newFastReader([]string{"zcat"}, r)
}

func newFastReader(command []string, r io.Reader) (*fastReader, error) {
	fr := fastReader{command: command}
	if err := fr.Reset(r); err != nil {
		return nil, err
	}
	return &fr, nil
}

func (fr *fastReader) Reset(r io.Reader) error {
	if fr.close != nil {
		fr.close()
	}

	cmd := exec.Command(fr.command[0], fr.command[1:]...)
	cmd.Stdin = r
	cmd.Stderr = &fr.stderr

	rpipe, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	// Don't get the process killed at CTRL+C
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
	err = cmd.Start()
	if err != nil {
		rpipe.Close()
		return err
	}

	fr.ReadCloser = rpipe
	fr.close = func() error {
		rpipe.Close()
		if err := cmd.Wait(); err != nil {
			if _, ok := err.(*exec.ExitError); ok {
				return errors.New(strings.TrimSpace(fr.stderr.String()))
			}
		}
		return err
	}
	return nil
}

func (fr *fastReader) Close() error {
	return fr.close()
}
