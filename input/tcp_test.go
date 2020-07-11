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
)

// this test checks that, given a reasonable amount of time (500ms) and under
// normal conditions, all log lines sent one by one on the TCP socket are
// received by the output.
func TestTCP1by1(t *testing.T) {
	toml := `
	[input]
	name="TCP"

	[output]
	name="RawRecorder"
	procs=1
	`
	c := baker.Components{
		Inputs:  []baker.InputDesc{TCPDesc},
		Outputs: []baker.OutputDesc{outputtest.RawRecorderDesc},
	}

	cfg, err := baker.NewConfigFromToml(strings.NewReader(toml), c)
	if err != nil {
		t.Fatal(err)
	}

	topology, err := baker.NewTopologyFromConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	topology.Start()
	// Give baker some time to start the tcp server
	time.Sleep(500 * time.Millisecond)

	const nlines = 4999 // num lines fed

	errc := make(chan error, 1)
	go func() {
		conn, err := net.Dial("tcp", ":6000")
		if err != nil {
			errc <- err
			return
		}
		defer conn.Close()

		w := gzip.NewWriter(conn)
		defer w.Close()

		for i := 0; i < nlines; i++ {
			buf := &bytes.Buffer{}
			l := baker.LogLine{FieldSeparator: baker.DefaultLogLineFieldSeparator}
			l.Set(1, []byte("field"))
			buf.Write(l.ToText(nil))
			buf.WriteByte('\n')
			if _, err := buf.WriteTo(w); err != nil {
				errc <- err
				return
			}
			if err := w.Flush(); err != nil {
				errc <- err
				return
			}
		}

		// Give us some time to send everything to baker, then stop
		time.Sleep(500 * time.Millisecond)
		topology.Stop()
		errc <- nil
	}()

	topology.Wait()
	if err := topology.Error(); err != nil {
		t.Fatalf("topology error: %v", err)
	}
	if err := <-errc; err != nil {
		t.Fatalf("error from sending goroutine: %v", err)
	}

	out := topology.Output[0].(*outputtest.Recorder)
	if len(out.Records) != nlines {
		t.Errorf("want %d log lines, got %d", nlines, len(out.Records))
	}
}

// this test checks that, given a reasonable amount of time (500ms) and under
// normal conditions, all log lines sent by chunk on the TCP socket are
// received by the output.
func TestTCPChunks(t *testing.T) {
	toml := `
	[input]
	name="TCP"

	[output]
	name="RawRecorder"
	procs=1
	`
	c := baker.Components{
		Inputs:  []baker.InputDesc{TCPDesc},
		Outputs: []baker.OutputDesc{outputtest.RawRecorderDesc},
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

	errc := make(chan error, 1)
	go func() {
		conn, err := net.Dial("tcp", ":6000")
		if err != nil {
			errc <- err
			return
		}
		defer conn.Close()

		w := gzip.NewWriter(conn)
		defer w.Close()

		buf := &bytes.Buffer{}

		for i := 0; i < nchunks; i++ {
			buf.Reset()
			for j := 0; j < chunksize; j++ {
				l := baker.LogLine{FieldSeparator: baker.DefaultLogLineFieldSeparator}
				l.Set(1, []byte("field"))
				buf.Write(l.ToText(nil))
				buf.WriteByte('\n')
				if _, err := buf.WriteTo(w); err != nil {
					errc <- err
					return
				}
			}
			// Send chunk
			if err := w.Flush(); err != nil {
				errc <- err
				return
			}
		}

		// Give us some time to send everything to baker, then stop
		time.Sleep(500 * time.Millisecond)
		topology.Stop()
		errc <- nil
	}()

	topology.Wait()
	if err := topology.Error(); err != nil {
		t.Fatalf("topology error: %v", err)
	}
	if err := <-errc; err != nil {
		t.Fatalf("error from sending goroutine: %v", err)
	}

	out := topology.Output[0].(*outputtest.Recorder)
	want := nchunks * chunksize
	if len(out.Records) != want {
		t.Errorf("want %d log lines, got %d", want, len(out.Records))
	}
}

// this test checks that when the topology is stopped while some log lines
// are sent in chunks via TCP, the number of log lines safely recevied by
// the output is a multiple of the chunk size (i.e whole chunks are received
// correctly).
func TestTCPStopChunk(t *testing.T) {
	toml := `
	[input]
	name="TCP"

	[output]
	name="RawRecorder"
	procs=1
	`
	c := baker.Components{
		Inputs:  []baker.InputDesc{TCPDesc},
		Outputs: []baker.OutputDesc{outputtest.RawRecorderDesc},
	}

	cfg, err := baker.NewConfigFromToml(strings.NewReader(toml), c)
	if err != nil {
		t.Fatal(err)
	}

	topology, err := baker.NewTopologyFromConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	topology.Start()
	// Give baker some time to start the tcp server
	time.Sleep(500 * time.Millisecond)

	const chunksz = 29 // number of log lines in a chunk

	errc := make(chan error, 1)
	go func() {
		conn, err := net.Dial("tcp", ":6000")
		if err != nil {
			errc <- err
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
				l := baker.LogLine{FieldSeparator: baker.DefaultLogLineFieldSeparator}
				l.Set(1, []byte("field"))
				buf.Write(l.ToText(nil))
				buf.WriteByte('\n')
				if _, err := buf.WriteTo(w); err != nil {
					errc <- err
					return
				}
			}
			if err := w.Flush(); err != nil {
				// this error is triggered by topology.Stop
				break
			}
		}
		errc <- nil
	}()

	topology.Wait()
	if err := topology.Error(); err != nil {
		t.Fatalf("topology error: %v", err)
	}
	if err := <-errc; err != nil {
		t.Fatalf("error from sending goroutine: %v", err)
	}

	out := topology.Output[0].(*outputtest.Recorder)
	// we should have received something at least
	if len(out.Records) == 0 {
		t.Errorf("len(out.Lines) = 0, want to receive something")
	}

	if len(out.Records)%chunksz != 0 {
		t.Errorf("len(out.Lines)= %d, want len(out.Lines)%%%d == 0, got %d", len(out.Records), chunksz, len(out.Records)%chunksz)
	}
}
