package input

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/AdRoll/baker"
	log "github.com/sirupsen/logrus"
)

var TCPDesc = baker.InputDesc{

	Name:   "TCP",
	New:    NewTCP,
	Config: &TCPConfig{},
	Help: "This input relies on a TCP connection to receive records in the usual format\n" +
		"Configure it with a host and port that you want to accept connection from.\n" +
		"By default it listens on port 6000 for any connection\n" +
		"It never exits.\n",
}

const (
	// gzipInput reads records in chunks, for maximizing speed. This is the
	// size of each chunk.
	tcpChunkBuffer = 128 * 1024

	// This is the expected maximum length of a single record. We still handle
	// longer lines, but with a slower code-path.
	tcpMaxLineLength = 4 * 1024
)

type TCPConfig struct {
	Listener string `help:"Host:Port to bind to"`
}

func (cfg *TCPConfig) fillDefaults() {
	if cfg.Listener == "" {
		cfg.Listener = ":6000"
	}
}

type TCP struct {
	Cfg *TCPConfig

	data     chan<- *baker.Data
	pool     sync.Pool
	numLines int64
	stop     int64
}

func NewTCP(cfg baker.InputParams) (baker.Input, error) {
	if cfg.DecodedConfig == nil {
		cfg.DecodedConfig = &TCPConfig{}
	}
	dcfg := cfg.DecodedConfig.(*TCPConfig)
	dcfg.fillDefaults()

	return &TCP{
		Cfg: dcfg,
		pool: sync.Pool{
			New: func() interface{} {
				return &baker.Data{Bytes: make([]byte, tcpChunkBuffer)}
			},
		},
	}, nil
}

func (s *TCP) Run(inch chan<- *baker.Data) error {
	var wg sync.WaitGroup
	s.setOutputChannel(inch)

	ctxLog := log.WithFields(log.Fields{"f": "Run"})

	addr, err := net.ResolveTCPAddr("tcp", s.Cfg.Listener)
	if err != nil {
		ctxLog.WithFields(log.Fields{"listener": s.Cfg.Listener}).Error("Can't resolve")
		return err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return err
	}
	defer l.Close()

	for atomic.LoadInt64(&s.stop) == 0 {
		l.SetDeadline(time.Now().Add(1 * time.Second))
		conn, err := l.AcceptTCP()
		if err != nil {
			if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
				continue
			}
			ctxLog.WithFields(log.Fields{"error": err}).Error("Error while accepting")
		}

		ctxLog.WithFields(log.Fields{"addr": conn.RemoteAddr()}).Info("Connected")
		wg.Add(1)
		go func(conn *net.TCPConn) {
			defer wg.Done()
			s.handleStream(conn)
		}(conn)
	}

	wg.Wait()
	return nil
}

func (s *TCP) setOutputChannel(data chan<- *baker.Data) {
	s.data = data
}

func (s *TCP) send(data *baker.Data) {
	nlines := int64(bytes.Count(data.Bytes, []byte{'\n'}))
	atomic.AddInt64(&s.numLines, nlines)

	s.data <- data
}

func (s *TCP) FreeMem(data *baker.Data) {
	data.Bytes = data.Bytes[:tcpChunkBuffer]
	s.pool.Put(data)
}

func (s *TCP) Stats() baker.InputStats {
	return baker.InputStats{
		NumProcessedLines: atomic.LoadInt64(&s.numLines),
	}
}

func (s *TCP) Stop() {
	atomic.StoreInt64(&s.stop, 1)
}

func (s *TCP) handleStream(conn *net.TCPConn) {
	defer conn.Close()
	ctxLog := log.WithFields(log.Fields{"f": "handleStream", "addr": conn.RemoteAddr()})

	// r, err := newFastGzReader(conn)
	r, err := gzip.NewReader(conn)
	if err != nil {
		ctxLog.WithError(err).Error("error initializing gzip")
		return
	}
	// defer r.Close()

	rbuf := bufio.NewReaderSize(r, tcpChunkBuffer)

	for atomic.LoadInt64(&s.stop) == 0 {
		bakerData := s.pool.Get().(*baker.Data)

		// Read a big chunk of data (but keeping tcpMaxLineLength
		// bytes available for completing the last line).
		n, err := rbuf.Read(bakerData.Bytes[:tcpChunkBuffer-tcpMaxLineLength])
		if err == io.EOF {
			bakerData.Bytes = bakerData.Bytes[:n]
			s.send(bakerData)
			break
		}

		if err != nil {
			ctxLog.WithError(err).Error("error reading stream")
			return
		}

		// We need to send a batch of complete lines to the filter
		// (sending truncated lines would generate parsing errors),
		// so we want to finish reading the last line we read until its
		// terminator.
		// NOTE: it might also happen that the chunk we just read
		// finished the file; so we check if the chunk ends with a
		// terminator, to avoid receiving a io.EOF from ReadBytes; EOFs
		// will be handled back when we begin the loop again.
		if bakerData.Bytes[n-1] != '\n' {
			endl, err := rbuf.ReadBytes('\n')
			if err != nil {
				ctxLog.WithError(err).Error("error searching newline")
				return
			}

			// If there is no space in the buffer to complete the
			// current line, we need to handle it differently.
			if n+len(endl) > tcpChunkBuffer {
				// Drop the initial part of the truncated line from the buffer
				lastn := n
				n = bytes.LastIndexByte(bakerData.Bytes[:n], '\n') + 1

				// Process the huge line by itself. Allocate a new buffer
				// from the pool, copy the initial part, and then concatenate
				// up to the endline
				bakerData2 := s.pool.Get().(*baker.Data)
				bakerData2.Meta = bakerData.Meta
				bakerData2.Bytes = append(bakerData2.Bytes[:0], bakerData.Bytes[n:lastn]...)
				bakerData2.Bytes = append(bakerData2.Bytes, endl...)
				s.send(bakerData2)
			} else {
				copy(bakerData.Bytes[n:], endl)
				n += len(endl)
			}
		}
		bakerData.Bytes = bakerData.Bytes[:n]
		s.send(bakerData)
	}
}
