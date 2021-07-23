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

const helpMsg = `This output writes the records into compressed files in a directory.
Files will be compressed using Gzip or Zstandard based on the filename extension in PathString.
The file names can contain placeholders that are populated by the output (see the keys help below).
When the special {{.Field0}} placeholder is used, then the user must specify the field name to
use as replacement in the fields configuration list.
The value of that field, extracted from each record, is used as replacement and, moreover, this
also means that each created file will contain only records with that same value for the field.
Note that, with this option, the FileWriter creates as many workers as the different values
of the field, and each one of these workers concurrently writes to a different file.
`

var FileWriterDesc = baker.OutputDesc{
	Name:   "FileWriter",
	New:    NewFileWriter,
	Config: &FileWriterConfig{},
	Raw:    true,
	Help:   helpMsg,
}

type FileWriterConfig struct {
	PathString           string        `help:"Template to describe location of the output directory: supports .Year, .Month, .Day, .Rotation and .Field0 (only with at least one field in [output.fields])."`
	RotateInterval       time.Duration `help:"Time after which data will be rotated. If -1, it will not rotate until the end." default:"60s"`
	ZstdCompressionLevel int           `help:"zstd compression level, ranging from 1 (best speed) to 19 (best compression)." default:"3"`
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

	for lldata := range input {
		wname := ""
		if w.useReplField {
			wname = lldata.Fields[0]
		}
		worker, ok := w.workers[wname]
		if !ok {
			// Unique UUID for the output processes
			uid := uuid.New().String()
			worker = newWorker(w.Cfg, wname, w.index, uid, upch)
			w.workers[wname] = worker
		}

		worker.Write(lldata.Record)

		atomic.AddInt64(&w.totaln, int64(1))
	}

	ctxlog.Info("FileWriter Terminating")

	// Concurrently close the workers, but with no more than 'NumCPU' goroutines.
	sem := make(chan struct{}, runtime.NumCPU())
	for i := range w.workers {
		i := i
		sem <- struct{}{}
		go func() {
			defer func() { <-sem }()
			err := w.workers[i].Close()
			if err != nil {
				ctxlog.WithError(err).Error("error when closing worker")
			}
		}()
	}

	return nil
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
	upch chan<- string

	cfg *FileWriterConfig

	pathTemplate   *template.Template
	replFieldValue string
	index          int
	uid            string
	rotateIdx      int64

	currentPath string
	fd          *os.File
	lock        sync.Mutex

	ticker  *time.Ticker
	writer  *bufio.Writer
	cwriter io.WriteCloser

	useZstd bool
}

const (
	fileWorkerChunkBuffer = 128 * 1024
)

func newWorker(cfg *FileWriterConfig, replFieldValue string, index int, uid string, upch chan<- string) *fileWorker {
	pathTemplate, err := template.New("fileWorkerType").Parse(cfg.PathString)
	if err != nil {
		panic(err.Error())
	}

	fw := &fileWorker{
		in:             make(chan []byte, 1),
		done:           make(chan struct{}),
		upch:           upch,
		cfg:            cfg,
		pathTemplate:   pathTemplate,
		replFieldValue: replFieldValue,
		index:          index,
		uid:            uid,
		useZstd:        false,
		rotateIdx:      0,
	}

	if strings.HasSuffix(cfg.PathString, ".zst") || strings.HasSuffix(cfg.PathString, ".zstd") {
		fw.useZstd = true
	}

	fw.Rotate()
	go fw.run()

	fw.ticker = time.NewTicker(cfg.RotateInterval)
	go func() {
		for range fw.ticker.C {
			fw.Rotate()
		}
	}()

	return fw
}

func (fw *fileWorker) makePath() string {
	now := time.Now().UTC()
	var doc bytes.Buffer

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

	err := fw.pathTemplate.Execute(&doc, replacementVars)
	if err != nil {
		panic(err.Error())
	}
	replacedPath := doc.String()
	dir := path.Dir(replacedPath)

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		os.MkdirAll(dir, 0777)
	}
	return replacedPath
}

func (fw *fileWorker) Rotate() {
	ctxLog := log.WithFields(log.Fields{"current": fw.currentPath, "idx": fw.index})

	ctxLog.Info("Rotating")
	oldPath := fw.currentPath
	fw.currentPath = fw.makePath()

	fd, err := os.OpenFile(fw.currentPath, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		ctxLog.Fatal("failed to rotate")
		panic(err)
	}
	w := bufio.NewWriterSize(fd, fileWorkerChunkBuffer)
	var cwriter io.WriteCloser
	if fw.useZstd {
		params := &zstd.WriterParams{
			CompressionLevel: fw.cfg.ZstdCompressionLevel,
			WindowLog:        fw.cfg.ZstdWindowLog,
		}
		cwriter = zstd.NewWriterParams(w, params)
	} else {
		cwriter, err = gzip.NewWriterLevel(w, gzip.BestSpeed)
	}
	if err != nil {
		ctxLog.WithError(err).Fatal("failed to rotate")
		panic(err)
	}

	fw.lock.Lock()
	defer fw.lock.Unlock()
	fw.closeall()
	fw.upload(oldPath)
	fw.fd = fd
	fw.writer = w
	fw.cwriter = cwriter
	fw.rotateIdx++
	ctxLog.Info("Rotated")
}

func (fw *fileWorker) upload(filepath string) {
	if filepath != "" {
		fw.upch <- filepath
	}
}

func (fw *fileWorker) Write(req []byte) {
	fw.in <- req
}

func (fw *fileWorker) Close() error {
	log.WithFields(log.Fields{"idx": fw.index}).Info("fileWorker closing")
	close(fw.in)
	<-fw.done
	return nil
}

func (fw *fileWorker) closeall() {
	if fw.cwriter != nil {
		fw.cwriter.Close()
	}
	if fw.writer != nil {
		fw.writer.Flush()
	}
	if fw.fd != nil {
		fw.fd.Close()
	}
}

func (fw *fileWorker) write(line []byte) error {
	fw.lock.Lock()
	defer fw.lock.Unlock()

	_, err := fw.cwriter.Write(line)
	fw.cwriter.Write([]byte("\n"))
	return err
}

func (fw *fileWorker) run() {
	for line := range fw.in {
		if err := fw.write(line); err != nil {
			log.WithError(err).Error("error writing to file")
		}
	}
	fw.ticker.Stop()
	fw.closeall()
	fw.upload(fw.currentPath)
	close(fw.done)
}
