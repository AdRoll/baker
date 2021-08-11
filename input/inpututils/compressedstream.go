package inpututils

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/klauspost/compress/gzip"
	log "github.com/sirupsen/logrus"
	zstd "github.com/valyala/gozstd"

	"github.com/AdRoll/baker"
)

const (
	// compressedInput reads records in chunks, for maximizing speed. This is the
	// size of each chunk.
	kChunkBuffer = 128 * 1024

	// This is the expected maximum length of a single record. We still handle
	// longer lines, but with a slower code-path.
	kMaxLineLength = 4 * 1024
)

type compressionType int

// List of compression types supported by this module.
const (
	gzipCompression compressionType = iota
	zstdCompression
)

// These keys identify values in the record Metadata cache
const (
	MetadataLastModified = "last_modified"
	MetadataURL          = "url"
)

// CompressedInput is a base for creating input components that processes
// multiple gzip or zstd-compressed logs coming from arbitrary sources.
//
// This class implements an internal queue of files (expressed by filenames)
// and instantiates a number of workers to process them. Subclasses can
// enqueue a file for processing through compressedInput.ProcessFile()
//
// It must be configured with an Opener function that is able to open
// a file given its filename and returns a io.ReadCloser instance for
// that file.
type CompressedInput struct {
	// atomically-accessed, keep on top for 64-bit alignment.
	stopping          int64
	numProcessedLines int64

	Opener func(fn string) (io.ReadCloser, int64, time.Time, *url.URL, error)
	Sizer  func(fn string) (int64, error)
	Done   chan bool

	files   chan string
	pool    sync.Pool
	data    chan<- *baker.Data
	stopNow chan struct{}

	stats *inputStats
}

type inputStats struct {
	// atomically-accessed, keep on top for 64-bit alignment.
	totalFiles     int64
	processedFiles int64
	totalSize      int64
	processedSize  int64

	beginTime time.Time
}

type inputStatsReader struct {
	// atomically-accessed, keep on top for 64-bit alignment.
	n int64

	s  *inputStats
	r  io.ReadCloser
	sz int64
}

func (i *inputStatsReader) Read(data []byte) (int, error) {
	n, err := i.r.Read(data)
	if i.sz != 0 {
		atomic.AddInt64(&i.n, int64(n))
		atomic.AddInt64(&i.s.processedSize, int64(n))
	}
	return n, err
}

func (i *inputStatsReader) Close() error {
	// If we knew the size of the file in advanced, and it's not been fully
	// processed for any rason (eg: truncated network connection), remove the
	// remaining size from the total size, so that the ETA estimation is
	// still correct.
	if i.sz != 0 {
		rem := i.sz - atomic.LoadInt64(&i.n)
		if rem >= 0 {
			atomic.AddInt64(&i.s.totalSize, -rem)
		}
	}
	atomic.AddInt64(&i.s.processedFiles, 1)

	return i.r.Close()
}

func newInputStats() *inputStats {
	s := new(inputStats)
	s.beginTime = time.Now()
	return s
}

func (s *inputStats) NewStatsReader(r io.ReadCloser, sz int64) io.ReadCloser {
	return &inputStatsReader{s: s, r: r, sz: sz}
}

func (s *inputStats) NewFile(sz int64) {
	atomic.AddInt64(&s.totalFiles, 1)
	atomic.AddInt64(&s.totalSize, sz)
}

func (s *inputStats) Stats() map[string]string {
	stats := make(map[string]string)

	processedSize := atomic.LoadInt64(&s.processedSize)
	totalSize := atomic.LoadInt64(&s.totalSize)
	if totalSize > 0 && processedSize > 0 {
		var estimatedEnd time.Time

		now := time.Now()
		elapsed := now.Sub(s.beginTime)
		speed := float64(elapsed) / float64(processedSize)
		estimatedEnd = now.Add(time.Duration(speed * float64(totalSize-processedSize)))

		eta := -time.Since(estimatedEnd)
		stats["ETA"] = fmt.Sprint(eta - (eta % time.Second))
	}

	stats["ProcessedFiles"] = fmt.Sprint(atomic.LoadInt64(&s.processedFiles))
	stats["TotalFiles"] = fmt.Sprint(atomic.LoadInt64(&s.totalFiles))

	return stats
}

func NewCompressedInput(opener func(fn string) (io.ReadCloser, int64, time.Time, *url.URL, error), sizer func(fn string) (int64, error), done chan bool) *CompressedInput {
	s := &CompressedInput{
		Opener:  opener,
		Sizer:   sizer,
		Done:    done,
		stats:   newInputStats(),
		files:   make(chan string, 1024),
		stopNow: make(chan struct{}),
		pool: sync.Pool{
			New: func() interface{} {
				return &baker.Data{Bytes: make([]byte, kChunkBuffer)}
			},
		},
	}

	// Start workers, that will read incoming files in the queue
	// and process them.
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.worker()
		}()
	}

	go func() {
		wg.Wait()
		close(s.Done)
	}()

	return s
}

func (s *CompressedInput) worker() {
	// Process incoming files on the s.files channel.
	// If the channel is closed, it means that we processed all the
	// channels that we should have processed, and we can safely exit.
	// If the s.stopNow channel is signaled, we need to abort
	// as soon as possible.
	for {
		select {
		case fn, ok := <-s.files:
			if !ok {
				// Channel is closed, we're done
				return
			}
			s.ParseFile(fn)
		case <-s.stopNow:
			return
		}
	}
}

func (s *CompressedInput) SetOutputChannel(data chan<- *baker.Data) {
	s.data = data
}

func (s *CompressedInput) send(data *baker.Data) {
	nlines := int64(bytes.Count(data.Bytes, []byte{'\n'}))
	atomic.AddInt64(&s.numProcessedLines, nlines)

	s.data <- data
}

// Enqueue a file for processing by compressedInput. This function must be called
// by subclasses to schedule processing a (gzip|zstd) logfile.
// The function just enqueues the file and exits, so it's normally fast,
// but might block if the backlog is bigger than internal channel size
// (default: 1024 files)
func (s *CompressedInput) ProcessFile(fn string) error {
	// Use the Sizer on the file to acquire the length
	sz, err := s.Sizer(fn)
	if err != nil {
		return err
	}
	s.stats.NewFile(sz)
	s.files <- fn
	return nil
}

// Signal compressedInput that we've finished enqueuing files, and it can exit
// whenever it has finished processing what was already enqueued. This can
// be used by an input which has a fixed set of files to process.
func (s *CompressedInput) NoMoreFiles() {
	close(s.files)
}

func (s *CompressedInput) Stop() {
	close(s.stopNow)
	atomic.StoreInt64(&s.stopping, 1)
}

func (s *CompressedInput) ParseFile(fn string) {
	if strings.HasSuffix(fn, ".zst") || strings.HasSuffix(fn, ".zstd") {
		s.parseFileTyped(fn, zstdCompression)
	} else {
		s.parseFileTyped(fn, gzipCompression)
	}
}

func (s *CompressedInput) parseFileTyped(fn string, comp compressionType) {

	ctx := log.WithFields(log.Fields{"f": "compressedInput.parseFile", "fn": fn})
	stream, sz, lastModified, url, err := s.Opener(fn)

	stream = s.stats.NewStatsReader(stream, sz)
	if err != nil {
		log.WithFields(log.Fields{"f": "compressedInput.parseFile", "fn": fn}).WithError(err).Error("Error while opening stream")
		return
	}
	defer stream.Close()

	var r io.Reader

	switch comp {
	case gzipCompression:
		if sz > 1000000 {
			rgz, err := newFastGzReader(stream)
			if err != nil {
				// Sometimes the fast gz reader fails to initialize due to
				// memory pressure. We'd still like to run so try the
				// slower (and less memory hungry) gzip.
				ctx.WithError(err).Error("error initializing fast gzip, will attempt slow gzip")
				r, err = gzip.NewReader(stream)
				if err != nil {
					ctx.WithError(err).Fatal("both fast and slow gzip readers failed to initialize")
					return
				}
			} else {
				defer rgz.Close()
				r = rgz
			}
		} else {
			rgz, err := gzip.NewReader(stream)
			if err != nil {
				ctx.WithError(err).Fatal("error initializing gzip")
				return
			}
			defer rgz.Close()
			r = rgz
		}
	case zstdCompression:
		rzst := zstd.NewReader(stream)
		defer rzst.Release()
		r = rzst
	default:
		ctx.WithError(err).Fatal("Unknown compression type specified.")
	}

	ctx.Info("begin reading")

	rbuf := bufio.NewReaderSize(r, kChunkBuffer)

	for atomic.LoadInt64(&s.stopping) == 0 {
		bakerData := s.pool.Get().(*baker.Data)
		bakerData.Meta = baker.Metadata{
			MetadataLastModified: lastModified,
			MetadataURL:          url,
		}

		// Read a big chunk of data (but keeping kMaxLineLength
		// bytes available for completing the last line).
		n, err := rbuf.Read(bakerData.Bytes[:kChunkBuffer-kMaxLineLength])
		if err == io.EOF {
			bakerData.Bytes = bakerData.Bytes[:n]
			s.send(bakerData)
			break
		}

		if err != nil {
			ctx.WithError(err).Error("error reading file")
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
				ctx.WithError(err).Error("error searching newline")
				return
			}

			// If there is no space in the buffer to complete the
			// current line, we need to handle it differently.
			if n+len(endl) > kChunkBuffer {
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

	ctx.Info("end")
}

func (s *CompressedInput) FreeMem(data *baker.Data) {
	data.Bytes = data.Bytes[:kChunkBuffer]
	s.pool.Put(data)
}

func (s *CompressedInput) Stats() baker.InputStats {
	return baker.InputStats{
		NumProcessedLines: atomic.LoadInt64(&s.numProcessedLines),
		CustomStats:       s.stats.Stats(),
	}
}
