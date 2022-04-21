package filter

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	log "github.com/sirupsen/logrus"

	"github.com/AdRoll/baker"
	"github.com/AdRoll/baker/pkg/zip_agnostic"
)

var ExternalMatchDesc = baker.FilterDesc{
	Name:   "ExternalMatch",
	New:    NewExternalMatch,
	Config: &ExternalMatchConfig{},
	Help:   "Discards records which fields matches values read from a CSV, which is possibly periodically refreshed. CSV files can be compressed (gz or zstd) or not.",
}

type ExternalMatchConfig struct {
	Region         string        `help:"AWS region to pass to S3 client (only for files with s3:// prefix)" default:"us-west-2"`
	Files          []string      `help:"URL(s) of CSV file(s) containing the strings to match (s3[n]:// or file://). If %s is present, it's replaced, at download time, with the result of calling time.Now().Format(DateTimeLayout)." required:"true"`
	DateTimeLayout string        `help:"Go date time string layout replacing %s in Files, evaluated just before downloading Files. See https://pkg.go.dev/time#Time.Format"`
	TimeSubtract   time.Duration `help:"Duration to subtract from time.Now() when evaluating DateTimeLayout. See https://pkg.go.dev/time#ParseDuration"`
	RefreshEvery   time.Duration `help:"Period at which Files are refreshed (downloaded again), if not set, Files are never refreshed"`
	CSVColumn      int           `help:"0-based index of the CSV column containing the values to consider" default:"0"`
	FieldName      string        `help:"Name of the record field to consider for the match" required:"true"`
	KeepOnMatch    bool          `help:"If true, keep records if field at FieldName matches any of the CSV values. If false, discard records if field matches any of the CSV values." default:"false"`

	fidx baker.FieldIndex // FieldIndex corresponding to FieldName
}

type ExternalMatch struct {
	cfg *ExternalMatchConfig

	mx     sync.RWMutex
	values map[string]struct{}

	numFilteredLines int64
	quit             chan struct{} // used to stop the 'refresh' goroutine
}

func (cfg *ExternalMatchConfig) fillDefaults() error {
	if cfg.Region == "" {
		cfg.Region = "us-west-2"
	}

	// Check that Files and DateTimeLayout are coherent.
	for _, u := range cfg.Files {
		isFormatted := strings.Contains(u, "%s")
		if isFormatted != (cfg.DateTimeLayout != "") {
			if cfg.DateTimeLayout != "" {
				return errors.New("DateTimeLayout is valid only if all strings in Files contain %s")
			}
			return errors.New("strings in Files may only contain %s if DateTimeLayout is set")
		}
	}

	// Check that TimeSubtract and DateTimeLayout are coherent.
	if (cfg.TimeSubtract != 0) && cfg.DateTimeLayout == "" {
		return errors.New("TimeSubtract is only valid if DateTimeLayout is set")
	}

	// Validate URLs
	for _, u := range cfg.evaluateURLs() {
		parsed, err := url.Parse(u)
		if err != nil {
			return err
		}
		switch parsed.Scheme {
		case "s3", "s3n", "file":
		default:
			return fmt.Errorf("%s: unsupported scheme %s", u, parsed.Scheme)
		}
	}

	if cfg.CSVColumn < 0 {
		return errors.New("negative CSVColumn")
	}

	return nil
}

// evaluateURLs evaluates urls using the current configuration..
func (cfg *ExternalMatchConfig) evaluateURLs() []string {
	var args []interface{}
	if cfg.DateTimeLayout != "" {
		now := time.Now().Add(-time.Duration(cfg.TimeSubtract))
		args = append(args, now.Format(cfg.DateTimeLayout))
	}

	ret := make([]string, 0, len(cfg.Files))
	for _, u := range cfg.Files {
		ret = append(ret, fmt.Sprintf(u, args...))
	}
	return ret
}

func NewExternalMatch(cfg baker.FilterParams) (baker.Filter, error) {
	dcfg := cfg.DecodedConfig.(*ExternalMatchConfig)
	if err := dcfg.fillDefaults(); err != nil {
		return nil, fmt.Errorf("ExternalMatch: invalid configuration: %v", err)
	}

	var found bool
	if dcfg.fidx, found = cfg.FieldByName(dcfg.FieldName); !found {
		return nil, fmt.Errorf("ExternalMatch: invalid configuration: no such field %v", dcfg.FieldName)
	}

	f := &ExternalMatch{cfg: dcfg, quit: make(chan struct{})}
	if err := f.updateValues(); err != nil {
		return nil, fmt.Errorf("ExternalMatch: failed loading values: %v", err)
	}

	if dcfg.RefreshEvery != 0 {
		go func() {
			tick := time.NewTicker(dcfg.RefreshEvery)
			for {
				select {
				// Terminate this goroutine. For now, this is only useful in
				// tests, to avoid race conditions at test cleanup.
				case <-f.quit:
					return
				case <-tick.C:
					if err := f.updateValues(); err != nil {
						log.WithError(err).Error("ExternalMatch: failed reloading values")
					}
				}
			}
		}()
	}

	return f, nil
}

// valuesFromCSV reads the CSV-formatted reader r and returns the set of values
// in the 0-based column index..
//
// Note: rows not having enough columns to extract the colIdx-th column are
// simply discarded.
func valuesFromCSV(r io.Reader, colIdx int) (map[string]struct{}, error) {
	csvReader := csv.NewReader(r)
	csvReader.ReuseRecord = true
	// -1 to not raise an error if rows do not all have the same number of fields.
	csvReader.FieldsPerRecord = -1

	values := make(map[string]struct{})
	for {
		row, err := csvReader.Read()
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return values, err
		}
		if colIdx >= len(row) {
			continue // omit row
		}
		values[row[colIdx]] = struct{}{}
	}
}

func (f *ExternalMatch) processURL(u string) (map[string]struct{}, error) {
	parsed, err := url.Parse(u)
	if err != nil {
		return nil, fmt.Errorf("error parsing URL: %v", err)
	}

	var r io.Reader

	log.WithField("url", u).Info("begin parsing file")
	switch parsed.Scheme {
	case "s3":
		sess, err := session.NewSession(&aws.Config{Region: aws.String(f.cfg.Region)})
		if err != nil {
			return nil, fmt.Errorf("error creating aws session: %v", err)
		}
		resp, err := s3.New(sess).GetObject(&s3.GetObjectInput{
			Bucket: aws.String(parsed.Host),
			Key:    aws.String(parsed.Path),
		})
		if err != nil {
			return nil, fmt.Errorf("error downloading file: %v", err)
		}

		defer resp.Body.Close()
		r = resp.Body

	case "file":
		path := filepath.Join(parsed.Host, parsed.Path) // On Windows Host contains the drive letter.
		f, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("error opening file: %v", err)
		}

		defer f.Close()
		r = f

	default:
		// scheme error should be caught during the validation of configuration.
		panic("unexpected scheme")
	}

	// Wrap the reader into a zip-agnostic reader to indifferently read gzip,
	// zstd or non compressed CSV data.
	zar, err := zip_agnostic.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("can't read from %s: %s", u, err)
	}
	defer zar.Close()

	return valuesFromCSV(zar, f.cfg.CSVColumn)
}

func (f *ExternalMatch) updateValues() error {
	values := make(map[string]struct{})

	for _, rurl := range f.cfg.evaluateURLs() {
		m, err := f.processURL(rurl)
		if err != nil {
			return err
		}
		log.WithFields(log.Fields{"url": rurl, "unique": len(m)}).Info("Successfully (re)loaded values from file")
		for k := range m {
			values[k] = struct{}{}
		}
	}

	f.mx.Lock()
	f.values = values
	f.mx.Unlock()
	return nil
}

func (f *ExternalMatch) Stats() baker.FilterStats {
	return baker.FilterStats{
		NumFilteredLines: atomic.LoadInt64(&f.numFilteredLines),
	}
}

func (f *ExternalMatch) Process(l baker.Record, next func(baker.Record)) {
	f.mx.RLock()
	_, ok := f.values[string(l.Get(f.cfg.fidx))]
	f.mx.RUnlock()

	if ok != f.cfg.KeepOnMatch {
		atomic.AddInt64(&f.numFilteredLines, 1)
		return
	}

	next(l)
}
