package input

import (
	"bytes"
	"compress/gzip"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/AdRoll/baker"
	"github.com/AdRoll/baker/output/outputtest"
	"github.com/AdRoll/baker/testutil"
)

// this test checks that, given a reasonable amount of time (500ms) and under
// normal conditions, all log lines sent one by one on the TCP socket are
// received by the output.
func TestIntegrationTCP1by1(t *testing.T) {
	testutil.InitLogger()
	toml := `
	[input]
	name="TCP"

	[output]
	name="Recorder"
	procs=1
	`
	c := baker.Components{
		Inputs:  []baker.InputDesc{TCPDesc},
		Outputs: []baker.OutputDesc{outputtest.RecorderDesc},
	}

	cfg, err := baker.NewConfigFromToml(strings.NewReader(toml), c)
	if err != nil {
		t.Error(err)
	}

	topology, err := baker.NewTopologyFromConfig(cfg)
	if err != nil {
		t.Error(err)
	}
	topology.Start()
	// Give baker some time to start the tcp server
	time.Sleep(500 * time.Millisecond)

	const nlines = 4999 // num lines fed

	go func() {
		conn, err := net.Dial("tcp", ":6000")
		if err != nil {
			t.Error(err)
			return
		}
		defer conn.Close()

		w := gzip.NewWriter(conn)
		defer w.Close()

		for i := 0; i < nlines; i++ {
			l := baker.LogLine{}
			buf := &bytes.Buffer{}
			buf.Write(l.ToText(nil))
			buf.WriteByte('\n')
			if _, err := buf.WriteTo(w); err != nil {
				t.Fatal(err)
			}
			if err := w.Flush(); err != nil {
				t.Fatal(err)
			}
		}

		// Give us some time to send everything to baker, then stop
		time.Sleep(500 * time.Millisecond)
		topology.Stop()
	}()

	topology.Wait()

	out := topology.Output[0].(*outputtest.Recorder)
	if len(out.LogLines) != nlines {
		t.Errorf("want %d log lines, got %d", nlines, len(out.LogLines))
	}
}

// this test checks that, given a reasonable amount of time (500ms) and under
// normal conditions, all log lines sent by chunk on the TCP socket are
// received by the output.
func TestIntegrationTCPChunks(t *testing.T) {
	testutil.InitLogger()
	toml := `
	[input]
	name="TCP"

	[output]
	name="Recorder"
	procs=1
	`
	c := baker.Components{
		Inputs:  []baker.InputDesc{TCPDesc},
		Outputs: []baker.OutputDesc{outputtest.RecorderDesc},
	}

	cfg, err := baker.NewConfigFromToml(strings.NewReader(toml), c)
	if err != nil {
		t.Error(err)
	}

	topology, err := baker.NewTopologyFromConfig(cfg)
	if err != nil {
		t.Error(err)
	}
	topology.Start()
	// Give baker some time to start the tcp server
	time.Sleep(500 * time.Millisecond)

	const (
		nchunks   = 100 // num lines fed
		chunksize = 37  // chunk size
	)

	go func() {
		conn, err := net.Dial("tcp", ":6000")
		if err != nil {
			t.Error(err)
			return
		}
		defer conn.Close()

		w := gzip.NewWriter(conn)
		defer w.Close()

		l := baker.LogLine{}
		buf := &bytes.Buffer{}

		for i := 0; i < nchunks; i++ {
			buf.Reset()
			for j := 0; j < chunksize; j++ {
				buf.Write(l.ToText(nil))
				buf.WriteByte('\n')
				if _, err := buf.WriteTo(w); err != nil {
					t.Fatal(err)
				}
			}
			// Send chunk
			if err := w.Flush(); err != nil {
				t.Fatal(err)
			}
		}

		// Give us some time to send everything to baker, then stop
		time.Sleep(500 * time.Millisecond)
		topology.Stop()
	}()

	topology.Wait()

	out := topology.Output[0].(*outputtest.Recorder)
	want := nchunks * chunksize
	if len(out.LogLines) != want {
		t.Errorf("want %d log lines, got %d", want, len(out.LogLines))
	}
}

// this test checks that when the topology is stopped while some log lines
// are sent in chunks via TCP, the number of log lines safely recevied by
// the output is a multiple of the chunk size (i.e whole chunks are received
// correctly).
func TestIntegrationTCPStopChunk(t *testing.T) {
	testutil.InitLogger()
	toml := `
	[input]
	name="TCP"

	[output]
	name="Recorder"
	procs=1
	`
	c := baker.Components{
		Inputs:  []baker.InputDesc{TCPDesc},
		Outputs: []baker.OutputDesc{outputtest.RecorderDesc},
	}

	cfg, err := baker.NewConfigFromToml(strings.NewReader(toml), c)
	if err != nil {
		t.Error(err)
	}

	topology, err := baker.NewTopologyFromConfig(cfg)
	if err != nil {
		t.Error(err)
	}
	topology.Start()
	// Give baker some time to start the tcp server
	time.Sleep(500 * time.Millisecond)

	const chunksz = 29 // number of log lines in a chunk

	go func() {
		conn, err := net.Dial("tcp", ":6000")
		if err != nil {
			t.Error(err)
			return
		}
		defer conn.Close()

		w := gzip.NewWriter(conn)
		defer w.Close()
		buf := &bytes.Buffer{}

		// Aynchronously stop topology after 250ms
		time.AfterFunc(250*time.Millisecond, func() {
			topology.Stop()
		})
		for {
			buf.Reset()
			for i := 0; i < chunksz; i++ {
				l := baker.LogLine{}
				buf.Write(l.ToText(nil))
				buf.WriteByte('\n')
				if _, err := buf.WriteTo(w); err != nil {
					t.Fatal(err)
				}
			}
			if err := w.Flush(); err != nil {
				// this error is triggered by topology.Stop
				break
			}
		}
	}()

	topology.Wait()

	out := topology.Output[0].(*outputtest.Recorder)
	// we should have received something at least
	if len(out.LogLines) == 0 {
		t.Errorf("len(out.Lines) = 0, want to receive something")
	}

	if len(out.LogLines)%chunksz != 0 {
		t.Errorf("len(out.Lines)= %d, want len(out.Lines)%%%d == 0, got %d", len(out.LogLines), chunksz, len(out.LogLines)%chunksz)
	}
}
