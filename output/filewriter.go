package output

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"text/template"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	zstd "github.com/valyala/gozstd"

	"github.com/AdRoll/baker"
)

const helpMsg = `This output writes serialized records into compressed files, gzip (.gz) or zstd
(.zst) depending on the file extension in PathString.

Generated files may be rotated if RotateInterval is set. PathString is used to
control the name of the generated files, it may contain placeholders. These
placeholders are evaluated each time a file is created, that is upon creation
of the output or everytime a rotation takes place.

Supported placeholders:
 - {{.Index}}     index of the current output process (see [output.procs]), 4 digits long
 - {{.Year}}      year (YYYY) at file creation
 - {{.Month}}     month number (MM) at file creation
 - {{.Day}}       day of the month number (DD) at file creation
 - {{.Hour}}      hour (HH) at file creation
 - {{.Minute}}    minute (MM) at file creation
 - {{.Second}}    second (SS) at file creation
 - {{.UUID}}      worker random UUID (v4), 36 chars long
 - {{.Rotation}}  rotation count, 6 digits long
 - {{.Field0}}    value of the first field provided in [output.fields] (only if present).
 
When choosing configuration values for your FileWriter, it's important to keep in mind
the following rules:

1. a file should only ever be accessed by a single worker at a time

If you use multiple output processes, you probably want to use {{.Index}} so that 
generated filenames are guaranteed to be different for each workers.

 2. rotation should never generate the same path twice
 
To avoid a file to be overwritten by its successor in the rotation, you should ensure
that 2 files generated at a distance of RotateInterval will have different filenames.
To ensure filenames are different, you should set RotateInterval to a duration that 
exceeds that of the time-based placeholder with the shortest span.

For example, the following is correct:

    PathString = "/path/to/file-{{.Hour}}-{{.Minute}}.log.gz" 
    RotateInterval = 5m

While the following is not correct:

    PathString = "/path/to/file-{{.Hour}}-{{.Minute}}.log.gz" 
    RotateInterval = 1s
`

var FileWriterDesc = baker.OutputDesc{
	Name:   "FileWriter",
	New:    NewFileWriter,
	Config: &FileWriterConfig{},
	Raw:    true,
	Help:   helpMsg,
}

type FileWriterConfig struct {
	PathString           string        `help:"Template describing names of the generated files. See top-level documentation for supported placeholders.."`
	RotateInterval       time.Duration `help:"Time interval between 2 successive file rotations. -1 disabled rotation." default:"60s"`
	ZstdCompressionLevel int           `help:"Zstd compression level, ranging from 1 (best speed) to 19 (best compression)." default:"3"`
	ZstdWindowLog        int           `help:"Enable zstd long distance matching. Increase memory usage for both compressor/decompressor. If more than 27 the decompressor requires special treatment. 0:disabled." default:"0"`
}

type FileWriter struct {
	Cfg *FileWriterConfig

	Fields []baker.FieldIndex
	totaln int64

	workers map[string]*fileWorker
	index   int

	useReplField bool
}

func NewFileWriter(cfg baker.OutputParams) (baker.Output, error) {
	log.WithFields(log.Fields{"fn": "NewFileWriter", "idx": cfg.Index}).Info("Initializing")

	dcfg := cfg.DecodedConfig.(*FileWriterConfig)
	dcfg.fillDefaults()

	fw := &FileWriter{
		Cfg:          dcfg,
		Fields:       cfg.Fields,
		workers:      make(map[string]*fileWorker),
		index:        cfg.Index,
		useReplField: strings.Contains(dcfg.PathString, "{{.Field0}}"),
	}

	if fw.useReplField && len(cfg.Fields) == 0 {
		return nil, errors.New("if {{.Field0}} is given, at least one field must be given in [output.fields]")
	}

	return fw, nil
}

func (w *FileWriter) Run(input <-chan baker.OutputRecord, upch chan<- string) error {
	ctxlog := log.WithFields(log.Fields{"idx": w.index})
	ctxlog.Info("FileWriter ready to log")


	var err error

	for lldata := range input {
		wname := ""
		if w.useReplField {
			wname = lldata.Fields[0]
		}
		worker, ok := w.workers[wname]
		if !ok {
			// Unique UUID for the output processes
			uid := uuid.New().String()
			worker, err = newWorker(w.Cfg, wname, w.index, uid, upch)
			if err != nil {
				// This error will be returned, but we'll try to cleanup the
				// potential other workers, not early exit.
				err = fmt.Errorf("FileWriter, can't create new worker: %s", err)
				break
			}
			w.workers[wname] = worker
		}

		worker.write(lldata.Record)

		atomic.AddInt64(&w.totaln, 1)
	}

	ctxlog.Info("FileWriter Terminating")

	// Concurrently close the workers, but with no more than 'NumCPU' goroutines.
	sem := make(chan struct{}, runtime.NumCPU())
	wg := sync.WaitGroup{}
	for i := range w.workers {
		i := i
		sem <- struct{}{}
		wg.Add(1)
		go func() {
			defer func() { <-sem; wg.Done() }()
			err := w.workers[i].Close()
			if err != nil {
				ctxlog.WithError(err).Error("error when closing worker")
			}
		}()
	}

	wg.Wait()

	return err
}

func (w *FileWriter) Stats() baker.OutputStats {
	return baker.OutputStats{
		NumProcessedLines: atomic.LoadInt64(&w.totaln),
	}
}

func (w *FileWriter) CanShard() bool {
	return false
}

func (cfg *FileWriterConfig) fillDefaults() {
	if cfg.PathString == "" {
		cfg.PathString = "/tmp/baker/ologs/logs/{{.Year}}/{{.Month}}/{{.Day}}/baker/{{.Year}}{{.Month}}{{.Day}}-{{.Hour}}{{.Minute}}{{.Second}}.{{.Index}}.log.gz"
	}
	var z time.Duration
	if cfg.RotateInterval == z {
		cfg.RotateInterval = 60 * time.Second
	}

	if cfg.ZstdCompressionLevel == 0 {
		cfg.ZstdCompressionLevel = 3
	}
}

// fileWorker manages writes to a file and its periodic rotation.
type fileWorker struct {
	in   chan []byte
	done chan struct{}

	cfg *FileWriterConfig

	pathTemplate   *template.Template
	replFieldValue string
	index          int
	uid            string
	rotateIdx      int64
}

const fileWorkerChunkBuffer = 128 * 1024

func newWorker(cfg *FileWriterConfig, replFieldValue string, index int, uid string, upch chan<- string) (*fileWorker, error) {
	ctxLog := log.WithFields(log.Fields{"output": "FileWriter", "idx": index})

	pathTemplate, err := template.New("fileWorkerType").Parse(cfg.PathString)
	if err != nil {
		panic(err.Error())
	}

	fw := &fileWorker{
		in:             make(chan []byte, 1),
		done:           make(chan struct{}),
		cfg:            cfg,
		pathTemplate:   pathTemplate,
		replFieldValue: replFieldValue,
		index:          index,
		uid:            uid,
		rotateIdx:      0,
	}

	useZstd := false
	if strings.HasSuffix(cfg.PathString, ".zst") || strings.HasSuffix(cfg.PathString, ".zstd") {
		useZstd = true
	}

	zstdParams := zstd.WriterParams{
		CompressionLevel: cfg.ZstdCompressionLevel,
		WindowLog:        cfg.ZstdWindowLog,
	}

	newFile := func(path string) (io.WriteCloser, error) {
		f, err := os.Create(path)
		if err != nil {
			return nil, err
		}

		bufw := bufio.NewWriterSize(f, fileWorkerChunkBuffer)

		var wc io.WriteCloser
		if useZstd {
			zstdw := zstd.NewWriterParams(bufw, &zstdParams)
			wc = makeWriteCloser(zstdw, zstdw.Close)
		} else {
			// Only way to for gzip.NewWriterLevel to fail is to pass an
			// incorrect compression level.
			wc, _ = gzip.NewWriterLevel(bufw, gzip.BestSpeed)
		}

		// Close the writers in order, as a stack of defer would do, with the
		// difference that they'll be closed when the called call Close, not
		// when the current function returns.
		close := func() error {
			if err := wc.Close(); err != nil {
				return fmt.Errorf("compression error: %s", err)
			}
			if err := bufw.Flush(); err != nil {
				return err
			}
			if err := f.Close(); err != nil {
				return err
			}
			return nil
		}

		return makeWriteCloser(wc, close), nil
	}

	curPath := fw.makePath()
	curw, err := newFile(curPath)
	if err != nil {
		return nil, fmt.Errorf("can't create file: %v", err)
	}

	go func() {
		var tick <-chan time.Time
		var ticker *time.Ticker
		if cfg.RotateInterval > 0 {
			ticker = time.NewTicker(cfg.RotateInterval)
			tick = ticker.C
		}

		defer func() {
			ctxLog.WithFields(log.Fields{"current": curPath}).Info("FileWriter worker terminating")

			// Close the last file and upload it.
			if err := curw.Close(); err != nil {
				ctxLog.WithError(err).WithField("current", curPath).Error("FileWriter worker error closing file")
			}
			upch <- curPath
			close(fw.done)
		}()

		for {
			select {
			case <-tick:
				// Perform rotation. Close, upload and swap curw with a newly
				// created file, after evaluating the path template.

				if err := curw.Close(); err != nil {
					ctxLog.WithError(err).WithField("current", curPath).Error("FileWriter worker error closing file")
				}

				upch <- curPath

				fw.rotateIdx++
				newPath := fw.makePath()
				ctxLog.WithFields(log.Fields{"current": curPath, "new": newPath}).Info("FileWriter worker file rotation")
				if curw, err = newFile(newPath); err != nil {
					ctxLog.WithError(err).WithField("current", curPath).Fatal("FileWriter worker can't create file")
				}
				curPath = newPath

			case line, ok := <-fw.in:
				if !ok {
					if ticker != nil {
						ticker.Stop()
					}
					return
				}
				if _, err := curw.Write(line); err != nil {
					log.WithError(err).Error("FileWriter worker error writing to file")
				}
				const linesep = "\n"
				if _, err := curw.Write([]byte(linesep)); err != nil {
					log.WithError(err).Error("FileWriter worker error writing to file")
				}
			}
		}
	}()

	return fw, nil
}

func (fw *fileWorker) write(req []byte) {
	fw.in <- req
}

func (fw *fileWorker) Close() error {
	fmt.Println("fileWorker.Close, closing in channel")
	close(fw.in)
	fmt.Println("fileWorker.Close, waiting for done channel to be closed")
	<-fw.done
	return nil
}

func (fw *fileWorker) makePath() string {
	now := time.Now().UTC()
	var buf bytes.Buffer

	replacementVars := map[string]string{
		"Index":    fmt.Sprintf("%04d", fw.index),
		"Year":     fmt.Sprintf("%04d", now.Year()),
		"Month":    fmt.Sprintf("%02d", now.Month()),
		"Day":      fmt.Sprintf("%02d", now.Day()),
		"Hour":     fmt.Sprintf("%02d", now.Hour()),
		"Minute":   fmt.Sprintf("%02d", now.Minute()),
		"Second":   fmt.Sprintf("%02d", now.Second()),
		"UUID":     fw.uid,
		"Rotation": fmt.Sprintf("%06d", fw.rotateIdx),
		"Field0":   fw.replFieldValue,
	}

	err := fw.pathTemplate.Execute(&buf, replacementVars)
	if err != nil {
		panic(err.Error())
	}
	dir := path.Dir(buf.String())
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		os.MkdirAll(dir, 0777)
	}
	return buf.String()
}

// makeWriteCloser converts an io.Writer and a Close function into a
// WriteCloser.
func makeWriteCloser(w io.Writer, close func() error) io.WriteCloser {
	return &writeCloser{Writer: w, close: close}
}

type writeCloser struct {
	io.Writer
	close func() error
}

func (wc *writeCloser) Close() error {
	return wc.close()
}
