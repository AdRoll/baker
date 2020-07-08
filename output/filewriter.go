package output

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"html/template"
	"io"
	"os"
	"path"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/google/uuid"
	zstd "github.com/valyala/gozstd"

	"github.com/AdRoll/baker"
	"github.com/AdRoll/baker/logger"
)

var FilesDesc = baker.OutputDesc{
	Name:   "Files",
	New:    NewFileWriter,
	Config: &FileWriterConfig{},
	Raw:    true,
	Help:   "This output writes the filtered log lines into typed files in a directory.\n",
}

type FileWriterConfig struct {
	PathString           string        `help:"Template to describe location of the output directory: supports .Type, .Year, .Month, .Day, .Region, .Instance, and .Rotation."`
	RotateInterval       time.Duration `help:"Time after which data will be rotated. If -1, it will not rotate until the end." default:"60s"`
	StagingPathString    string        `help:"Staging directory for the upload functionality"`
	Region               string        `help:"Replaces {{.Region}} in PathString. If empty it is set to 'region' from EC2 Metadata."`
	InstanceID           string        `help:"Replaces {{.Instance}} in PathString. If empty it is set to 'instance-id' from EC2 Metadata."`
	ZstdCompressionLevel int           `help:"zstd compression level, ranging from 1 (best speed) to 19 (best compression)." default:"3"`
	ZstdWindowLog        int           `help:"Enable zstd long distance matching. Increase memory usage for both compressor/decompressor. If more than 27 the decompressor requires special treatment. 0:disabled." default:"0"`
}

func (cfg *FileWriterConfig) fillDefaults() {
	if cfg.PathString == "" {
		cfg.PathString = "/tmp/baker/ologs/logs/{{.Year}}/{{.Month}}/{{.Day}}/{{.Type}}-baker-{{.Instance}}-{{.Region}}/{{.Type}}-{{.Year}}{{.Month}}{{.Day}}-{{.Hour}}{{.Minute}}{{.Second}}.{{.Index}}.log.gz"
	}
	var z time.Duration
	if cfg.RotateInterval == z {
		cfg.RotateInterval = 60 * time.Second
	}

	if strings.Contains(cfg.PathString, "{{.Region}}") && cfg.Region == "" {
		md := ec2metadata.New(session.New())

		region, err := md.Region()
		if err != nil {
			logger.Log.Error("Couldn't fetch region. ", err)
			region = ""
		}
		cfg.Region = region
	}

	if strings.Contains(cfg.PathString, "{{.Instance}}") && cfg.InstanceID == "" {
		md := ec2metadata.New(session.New())

		instanceid, err := md.GetMetadata("instance-id")
		if err != nil {
			logger.Log.Error("Couldn't fetch instance-id. ", err)
			instanceid = ""
		}

		cfg.InstanceID = instanceid
	}

	if cfg.ZstdCompressionLevel == 0 {
		cfg.ZstdCompressionLevel = 3
	}
}

type FileWriter struct {
	Cfg *FileWriterConfig

	Fields []baker.FieldIndex
	totaln int64

	workers map[string]*fileWorker
	index   int
}

func makeFileWriterPath(p *template.Template, t string, idx int, region, instanceid, uid string) string {
	now := time.Now().UTC()
	var doc bytes.Buffer

	replacementVars := map[string]string{
		"Type":     t,
		"Index":    fmt.Sprintf("%04d", idx),
		"Year":     fmt.Sprintf("%04d", now.Year()),
		"Month":    fmt.Sprintf("%02d", now.Month()),
		"Day":      fmt.Sprintf("%02d", now.Day()),
		"Hour":     fmt.Sprintf("%02d", now.Hour()),
		"Minute":   fmt.Sprintf("%02d", now.Minute()),
		"Second":   fmt.Sprintf("%02d", now.Second()),
		"Instance": instanceid,
		"UUID":     uid,
		"Region":   region,
	}
	err := p.Execute(&doc, replacementVars)
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

func NewFileWriter(cfg baker.OutputParams) (baker.Output, error) {
	logger.Log.Info("Initializing. fn=NewFileWriter, idx=", cfg.Index)

	if cfg.DecodedConfig == nil {
		cfg.DecodedConfig = &FileWriterConfig{}
	}
	dcfg := cfg.DecodedConfig.(*FileWriterConfig)
	dcfg.fillDefaults()

	return &FileWriter{
		Cfg:     dcfg,
		Fields:  cfg.Fields,
		workers: make(map[string]*fileWorker),
		index:   cfg.Index,
	}, nil
}

func (w *FileWriter) Run(input <-chan baker.OutputLogLine, upch chan<- string) {
	logger.Log.Info("FileWriter ready to log. idx=", w.index)

	for lldata := range input {
		if len(lldata.Line) < 3 {
			continue // bad line
		}
		linetype := string(lldata.Line[:3])

		worker, ok := w.workers[linetype]
		if !ok {
			// Unique UUID for the output processes
			uid := uuid.New().String()
			worker = newWorker(w.Cfg, linetype, w.index, uid, upch)
			w.workers[linetype] = worker
		}

		worker.Write(lldata.Line)

		atomic.AddInt64(&w.totaln, int64(1))
	}

	logger.Log.Info("FileWriter Terminating. idx=", w.index)
	for _, worker := range w.workers {
		worker.Close()
	}
	for _, worker := range w.workers {
		worker.Wait()
	}
}

func (w *FileWriter) Stats() baker.OutputStats {
	return baker.OutputStats{
		NumProcessedLines: atomic.LoadInt64(&w.totaln),
	}
}

func (w *FileWriter) CanShard() bool {
	return false
}

// Internal object only.
// a fileWorker instance will be responsible for
// managing writing to a file including rotating
// it periodically.

type fileWorker struct {
	in   chan []byte
	done chan bool
	upch chan<- string

	cfg *FileWriterConfig

	pathTemplate *template.Template
	linetype     string
	index        int
	uid          string
	region       string
	rotateIdx    int64

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

func newWorker(cfg *FileWriterConfig, linetype string, index int, uid string, upch chan<- string) *fileWorker {
	pathTemplate, err := template.New("fileWorkerType").Parse(cfg.PathString)
	if err != nil {
		panic(err.Error())
	}

	fw := &fileWorker{
		in:           make(chan []byte, 1),
		done:         make(chan bool, 1),
		upch:         upch,
		cfg:          cfg,
		pathTemplate: pathTemplate,
		linetype:     linetype,
		index:        index,
		uid:          uid,
		region:       cfg.Region,
		useZstd:      false,
		rotateIdx:    0,
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
		"Type":     fw.linetype,
		"Index":    fmt.Sprintf("%04d", fw.index),
		"Year":     fmt.Sprintf("%04d", now.Year()),
		"Month":    fmt.Sprintf("%02d", now.Month()),
		"Day":      fmt.Sprintf("%02d", now.Day()),
		"Hour":     fmt.Sprintf("%02d", now.Hour()),
		"Minute":   fmt.Sprintf("%02d", now.Minute()),
		"Second":   fmt.Sprintf("%02d", now.Second()),
		"Instance": fw.cfg.InstanceID,
		"UUID":     fw.uid,
		"Region":   fw.region,
		"Rotation": fmt.Sprintf("%06d", fw.rotateIdx),
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
	ctxLog := fmt.Sprintf("current=%s, idx=%d", fw.currentPath, fw.index)

	logger.Log.Info("Rotating. ", ctxLog)
	oldPath := fw.currentPath
	fw.currentPath = fw.makePath()

	fd, err := os.OpenFile(fw.currentPath, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		logger.Log.Fatal("failed to rotate", ctxLog)
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
		logger.Log.Fatalf("failed to rotate: %v %s", err, ctxLog)
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
	logger.Log.Info("Rotated", ctxLog)
}

func (fw *fileWorker) upload(filepath string) {
	if filepath != "" {
		fw.upch <- filepath
	}
}

func (fw *fileWorker) Write(req []byte) {
	fw.in <- req
}

func (fw *fileWorker) Close() {
	logger.Log.Info("fileWorker closing. idx=", fw.index)
	close(fw.in)
}

func (fw *fileWorker) Wait() bool {
	return <-fw.done
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
			logger.Log.Error("error writing to file. ", err)
		}
	}
	fw.ticker.Stop()
	fw.closeall()
	fw.upload(fw.currentPath)
	fw.done <- true
}
