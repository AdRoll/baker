package input

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/AdRoll/baker"
	"github.com/AdRoll/baker/input/inpututils"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	log "github.com/sirupsen/logrus"
)

// TODO[open-source] Evaluate whether the last_modified HTTP header (that goes to the metadata field)
// is always present, in particular http(s) header could be absent

var ListDesc = baker.InputDesc{
	Name:   "List",
	New:    NewList,
	Config: &ListConfig{},
	Help: "This input fetches logs from a predefined list of local or remote sources. The \"Files\"\n" +
		"configuration variable is a list of \"file specifiers\". Each \"file specifier\" can be:\n\n" +
		"  * A local file path on the filesystem: the log file at that path will be processed\n" +
		"  * A HTTP/HTTPS URL: the log file at that URL will be downloaded and processed\n" +
		"  * A S3 URL: the log file at that URL that will be downloaded and processed\n" +
		"  * \"@\" followed by a local path pointing to a file: the file is expected to be a text file\n" +
		"    and each line will be read and parsed as a \"file specifier\"\n" +
		"  * \"@\" followed by a HTTP/HTTPS URL: the text file pointed by the URL will be downloaded,\n" +
		"    and each line will be read and parsed as a \"file specifier\"\n" +
		"  * \"@\" followed by a S3 URL pointing to a file: the text file pointed by the URL will be\n" +
		"    downloaded, and each line will be read and parsed as a \"file specifier\"\n" +
		"  * \"@\" followed by a local path pointing to a directory (must end with a slash): the directory will be recursively\n" +
		"    walked, and all files matching the \"MatchPath\" option regexp will be processed as logfiles\n" +
		"  * \"@\" followed by a S3 URL pointing to a directory: the directory on S3 will be recursively\n" +
		"    walked, and all files matching the \"MatchPath\" option regexp will be processed as logfiles\n" +
		"  * \"-\": the contents of a log file will be read from stdin and processed\n" +
		"  * \"@-\": each line read from stdin will be parsed as a \"file specifier\"\n\n" +
		"All records produced by this input contain 2 metadata values:\n" +
		"  * url: the files that originally contained the record\n" +
		"  * last_modified: the last modification datetime of the above file\n",
}

var stdin = os.Stdin // for tests

type ListConfig struct {
	Files     []string `help:"List of log-files, directories and/or list-files to process" default:"[\"-\"]"`
	MatchPath string   `help:"regexp to filter files in specified directories" default:".*\\.log\\.gz"`
	Region    string   `help:"AWS Region for fetching from S3" default:"us-west-2"`
}

func (cfg *ListConfig) fillDefaults() {
	if cfg.MatchPath == "" {
		cfg.MatchPath = ".*\\.log\\.gz"
	}

	if cfg.Region == "" {
		cfg.Region = "us-west-2"
	}

	if len(cfg.Files) == 0 {
		cfg.Files = []string{"-"}
	}
}

type List struct {
	ci        *inpututils.CompressedInput
	svc       *s3.S3
	Cfg       *ListConfig
	matchPath *regexp.Regexp
	fatalErr  atomic.Value
	stopOnce  sync.Once
}

func (s *List) openFile(fn string, sizeOnly bool) (io.ReadCloser, int64, time.Time, *url.URL, error) {
	if fn == "-" {
		return stdin, 0, time.Unix(0, 0), nil, nil
	}

	u, err := url.Parse(fn)
	if err != nil {
		// NOTE: raw paths are parsed with u.Scheme=""
		s.setFatalErr(err)
		return nil, 0, time.Unix(0, 0), nil, err
	}

	switch u.Scheme {
	case "", "file":
		if fi, err := os.Stat(u.Path); err != nil {
			s.setFatalErr(err)
			return nil, 0, time.Unix(0, 0), u, err
		} else {
			f, err := os.Open(u.Path)
			if err != nil {
				s.setFatalErr(err)
				return nil, fi.Size(), fi.ModTime(), u, err
			}
			return f, fi.Size(), fi.ModTime(), u, err
		}
	case "s3":
		if sizeOnly {
			resp, err := s.svc.HeadObject(&s3.HeadObjectInput{
				Bucket: aws.String(u.Host),
				Key:    aws.String(u.Path),
			})
			if err != nil {
				err := fmt.Errorf("error opening %q: %v", fn, err)
				s.setFatalErr(err)
				return nil, 0, time.Unix(0, 0), u, err
			}
			return nil, *resp.ContentLength, *resp.LastModified, u, nil
		} else {
			resp, err := s.svc.GetObject(&s3.GetObjectInput{
				Bucket: aws.String(u.Host),
				Key:    aws.String(u.Path),
			})
			if err != nil {
				err := fmt.Errorf("error opening %q: %v", fn, err)
				s.setFatalErr(err)
				return nil, 0, time.Unix(0, 0), u, err
			}
			return resp.Body, *resp.ContentLength, *resp.LastModified, u, nil
		}
	case "http", "https":
		resp, err := http.Get(fn)
		if err != nil {
			s.setFatalErr(err)
			return nil, 0, time.Unix(0, 0), u, err
		}
		ssize := resp.Header.Get("Content-Length")
		size, _ := strconv.ParseInt(ssize, 10, 64)
		sLastModified := resp.Header.Get("Last-Modified")
		lastModified, _ := time.Parse("Mon, 02 Jan 2006 15:04:05 GMT", sLastModified)

		return resp.Body, size, lastModified, u, nil

	default:
		err := fmt.Errorf("unknown schema: %q", u.Scheme)
		s.setFatalErr(err)
		return nil, 0, time.Unix(0, 0), u, err
	}
}

func (s *List) ProcessDirectory(dir string, matchPath *regexp.Regexp) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err == nil && matchPath.MatchString(path) {
			s.ci.ProcessFile(path)
		}
		return nil
	})
}

func NewList(cfg baker.InputParams) (baker.Input, error) {
	inpututils.SetGCPercentIfNotSet(800)

	if cfg.DecodedConfig == nil {
		cfg.DecodedConfig = &ListConfig{}
	}
	dcfg := cfg.DecodedConfig.(*ListConfig)
	dcfg.fillDefaults()

	s3end := s3.New(session.New(&aws.Config{Region: aws.String(dcfg.Region)}))
	l := &List{
		svc: s3end,
		Cfg: dcfg,
	}

	opener := func(fn string) (io.ReadCloser, int64, time.Time, *url.URL, error) {
		blob, sz, lastModified, url, err := l.openFile(fn, false)
		return blob, sz, lastModified, url, err
	}
	sizer := func(fn string) (int64, error) {
		_, sz, _, _, err := l.openFile(fn, true)
		return sz, err
	}

	l.ci = inpututils.NewCompressedInput(opener, sizer, make(chan bool, 1))
	l.matchPath = regexp.MustCompile(dcfg.MatchPath)

	return l, nil
}

// Set this error as fatal: it will make List stop doing any processing,
// and Run() will report this error as return value
func (s *List) setFatalErr(err error) {
	s.stopOnce.Do(func() {
		// ci.Stop() can be called only once (or it will panic),
		// while there are many reasons why we'd go through a fatal error
		s.fatalErr.Store(err)
		s.ci.Stop()
	})
}

func (s *List) processListFile(f io.ReadCloser) {

	// Parse the list file line by line, in a way that allows
	// us to also check the status of the s.ci.Done channel
	// to abort.
	// This handles the case in which we're reading from stdin
	// which is possibly never closed, but we still want to abort
	// (eg: CTRL+C).
	lines := make(chan string, 8)

	go func() {
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}
			err := scanner.Err()
			if err != nil {
				log.WithError(err).Fatal("Failed to scan list input file.")
			}
			lines <- line
		}
		err := scanner.Err()
		if err != nil {
			log.WithError(err).Fatal("Failed to scan list input file.")
		}
		close(lines)
	}()

	for {
		select {
		case line, ok := <-lines:
			if !ok {
				return
			}

			s.processFileOrList(line)
		case <-s.ci.Done:
			return
		}
	}
}

func (s *List) processList(fn string) error {
	if fn == "-" {
		s.processListFile(stdin)
		return nil
	}

	u, err := url.Parse(fn)
	if err != nil {
		// NOTE: raw paths are parsed with u.Scheme=""
		return err
	}

	switch u.Scheme {
	case "", "file":
		// A list file on the local disk can be either a real file
		// or a directory (whose contents will be processed)
		if fi, err := os.Stat(u.Path); err != nil {
			return err
		} else if fi.IsDir() {
			return filepath.Walk(u.Path, func(path string, info os.FileInfo, err error) error {
				if err == nil && s.matchPath.MatchString(path) {
					s.ci.ProcessFile(path)
				}
				return nil
			})
		} else {
			f, err := os.Open(u.Path)
			if err != nil {
				return err
			}
			s.processListFile(f)
			return nil
		}

	case "s3":
		if u.Path[len(u.Path)-1:] == "/" {
			// ListObjectsV2Input prefix must not start with /
			prefix := strings.TrimLeft(u.Path, "/")

			paths := make(chan string)
			errCh := make(chan error)

			go func() {
				defer close(paths)

				var nextToken *string
				input := &s3.ListObjectsV2Input{
					Bucket:  aws.String(u.Host),
					Prefix:  aws.String(prefix),
					MaxKeys: aws.Int64(1000), // 1000 is the max value
				}
				for {
					if nextToken != nil {
						input.ContinuationToken = nextToken
					}

					resp, err := s.svc.ListObjectsV2(input)
					if err != nil {
						errCh <- err
						return
					}

					for _, obj := range resp.Contents {
						path := *obj.Key
						if s.matchPath.MatchString(path) {
							paths <- path
						}
					}

					if *(resp.IsTruncated) == false {
						return
					}
					nextToken = resp.NextContinuationToken
				}
			}()

			for {
				select {
				case err := <-errCh:
					return err
				case line, ok := <-paths:
					if !ok {
						return nil
					}
					s.ci.ProcessFile(fmt.Sprintf("s3://%s/%s", u.Host, line))
				case <-s.ci.Done:
					return nil
				}
			}
		} else {
			resp, err := s.svc.GetObject(&s3.GetObjectInput{
				Bucket: aws.String(u.Host),
				Key:    aws.String(u.Path),
			})
			if err != nil {
				return err
			}

			s.processListFile(resp.Body)
			resp.Body.Close()
			return nil
		}

	case "http", "https":
		resp, err := http.Get(u.Path)
		if err != nil {
			return err
		}
		s.processListFile(resp.Body)
		resp.Body.Close()
		return nil

	default:
		return fmt.Errorf("unknown scheme: %q", u.Scheme)
	}
}

func (s *List) processFileOrList(f string) {
	if f[0] == '@' {
		// List file
		if err := s.processList(f[1:]); err != nil {
			s.setFatalErr(fmt.Errorf("error parsing list file %q: %v", f[1:], err))
		}
	} else {
		// Regular file, just enqueue for processing
		s.ci.ProcessFile(f)
	}
}

func (s *List) Run(inch chan<- *baker.Data) error {
	s.ci.SetOutputChannel(inch)

	for _, f := range s.Cfg.Files {
		s.processFileOrList(f)
		if ferr := s.fatalErr.Load(); ferr != nil {
			// Even if we've got a fatal error, wait for
			// completion of compressedStream so that we're sure
			// all workers are exited. Otherwise, we might
			// cauase race-conditions in callers because
			// we haven't actually finished pushing things
			// into the output channel.
			break
		}
	}

	// Now wait until we've finished processing all the files
	s.ci.NoMoreFiles()
	<-s.ci.Done

	log.WithFields(log.Fields{"f": "List.Run"}).Info("terminating")
	if ferr := s.fatalErr.Load(); ferr != nil {
		return ferr.(error)
	}
	return nil
}

func (s *List) FreeMem(data *baker.Data) {
	s.ci.FreeMem(data)
}

func (s *List) Stats() baker.InputStats {
	return s.ci.Stats()
}

func (s *List) Stop() {
	s.setFatalErr(errors.New("abort requested"))
}
