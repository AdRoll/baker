package output

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/AdRoll/baker"
	"github.com/bmizerany/perks/quantile"
)

var StatsDesc = baker.OutputDesc{
	Name:   "Stats",
	New:    NewStats,
	Config: &StatsConfig{},
	Raw:    true,
	Help: "Compute various distributions of the records it " +
		"receives and dumps that to CSV. It computes the " +
		"distribution of record by size and the distribution " +
		"of the values of certain fields\n",
}

type StatsConfig struct {
	CountEmptyFields bool   `help:"Whether fields with empty values are counted or not" default:"false"`
	CSVPath          string `help:"Path of the CSV file to create" default:"stats.csv"`
	TimestampField   string `help:"Name of a field containing a POSIX timestamp (in seconds) used to build the times stats" required:"true"`
}

func (cfg *StatsConfig) fillDefaults() {
	if cfg.CSVPath == "" {
		cfg.CSVPath = "stats.csv"
	}
}

type sizeStats struct {
	totbytes uint64
	errs     uint64 // malformed lines
	qt       *quantile.Stream

	smallest, biggest uint32
}

func (s *sizeStats) add(size uint32, ll baker.Record, valid bool) {
	if !valid {
		s.errs++
		return
	}

	if size > s.biggest {
		s.biggest = size
	}
	if size < s.smallest {
		s.smallest = size
	}
	s.qt.Insert(float64(size))
	s.totbytes += uint64(size)
}

func (s *sizeStats) print(w io.Writer) error {
	csvw := csv.NewWriter(w)

	err := csvw.Write([]string{
		"num samples (log lines)",
		"errors",
		"total bytes",
		"smallest",
		"1st %%ile", "5th %%ile", "10th %%ile", "25th %%ile", "50th %%ile", "75th %%ile", "90th %%ile", "95th %%ile", "99th %%ile",
		"biggest"})
	if err != nil {
		return err
	}

	err = csvw.Write([]string{
		fmt.Sprintf("%v", s.qt.Count()),
		fmt.Sprintf("%v", s.errs),
		fmt.Sprintf("%v", s.totbytes),
		fmt.Sprintf("%v", s.smallest),
		fmt.Sprintf("%v", s.qt.Query(0.01)),
		fmt.Sprintf("%v", s.qt.Query(0.05)),
		fmt.Sprintf("%v", s.qt.Query(0.1)),
		fmt.Sprintf("%v", s.qt.Query(0.25)),
		fmt.Sprintf("%v", s.qt.Query(0.5)),
		fmt.Sprintf("%v", s.qt.Query(0.75)),
		fmt.Sprintf("%v", s.qt.Query(0.90)),
		fmt.Sprintf("%v", s.qt.Query(0.95)),
		fmt.Sprintf("%v", s.qt.Query(0.99)),
		fmt.Sprintf("%v", s.biggest),
	})
	if err != nil {
		return err
	}

	csvw.Flush()
	return csvw.Error()
}

type fieldStats struct {
	m       map[string]uint
	field   baker.FieldIndex
	empties bool // count empty fields?
}

func (s *fieldStats) add(ll baker.Record) {
	b := ll.Get(s.field)
	if !s.empties && b == nil {
		return
	}

	s.m[string(b)]++
}

func (s *fieldStats) print(w io.Writer, fieldNames []string) error {
	var smallest, biggest uint = math.MaxUint32, 0

	qt := quantile.NewTargeted(0.01, 0.05, 0.1, 0.25, 0.5, 0.75, 0.90, 0.95, 0.99)
	for _, freq := range s.m {
		qt.Insert(float64(freq))

		if freq > biggest {
			biggest = freq
		}

		if freq < smallest {
			smallest = freq
		}
	}

	csvw := csv.NewWriter(w)
	if err := csvw.Write(
		[]string{fmt.Sprintf(
			"num samples (%s)", fieldNames[s.field]),
			"smallest",
			"1st %%ile", "5th %%ile", "10th %%ile", "25th %%ile", "50th %%ile", "75th %%ile", "90th %%ile", "95th %%ile", "99th %%ile",
			"biggest"}); err != nil {
		return err
	}

	if err := csvw.Write([]string{
		fmt.Sprintf("%v", qt.Count()),
		fmt.Sprintf("%v", smallest),
		fmt.Sprintf("%v", qt.Query(0.01)),
		fmt.Sprintf("%v", qt.Query(0.05)),
		fmt.Sprintf("%v", qt.Query(0.1)),
		fmt.Sprintf("%v", qt.Query(0.25)),
		fmt.Sprintf("%v", qt.Query(0.5)),
		fmt.Sprintf("%v", qt.Query(0.75)),
		fmt.Sprintf("%v", qt.Query(0.90)),
		fmt.Sprintf("%v", qt.Query(0.95)),
		fmt.Sprintf("%v", qt.Query(0.99)),
		fmt.Sprintf("%v", biggest),
	}); err != nil {
		return err
	}

	csvw.Flush()
	return csvw.Error()
}

type timestampStats struct {
	nerrors     int64 // count malformed timestamps
	nempties    int64 // count empty timestamps
	first, last int64 // time range bounds
	qt          *quantile.Stream
	fieldIdx    baker.FieldIndex // The index of the timestamp field in the record
}

func (s *timestampStats) add(ll baker.Record) {
	b := ll.Get(s.fieldIdx)
	if b == nil {
		s.nempties++
		return
	}

	ts, err := strconv.Atoi(string(b))
	if err != nil {
		s.nerrors++
		return
	}

	if int64(ts) < s.first {
		s.first = int64(ts)
	}
	if int64(ts) > s.last {
		s.last = int64(ts)
	}

	s.qt.Insert(float64(ts))
	return
}

func (s *timestampStats) print(w io.Writer) error {
	csvw := csv.NewWriter(w)
	if err := csvw.Write(
		[]string{
			"num timestamps (valid+invalid+empty)",
			"num errors",
			"num empty",
			"first",
			"1st %%ile", "5th %%ile", "10th %%ile", "25th %%ile", "50th %%ile", "75th %%ile", "90th %%ile", "95th %%ile", "99th %%ile",
			"last",
		}); err != nil {
		return err
	}

	if err := csvw.Write([]string{
		fmt.Sprintf("%v", s.qt.Count()),
		fmt.Sprintf("%v", s.nerrors),
		fmt.Sprintf("%v", s.nempties),
		fmt.Sprintf("%v", time.Unix(s.first, 0).UTC()),
		fmt.Sprintf("%v", time.Unix(int64(s.qt.Query(0.01)), 0).UTC()),
		fmt.Sprintf("%v", time.Unix(int64(s.qt.Query(0.05)), 0).UTC()),
		fmt.Sprintf("%v", time.Unix(int64(s.qt.Query(0.1)), 0).UTC()),
		fmt.Sprintf("%v", time.Unix(int64(s.qt.Query(0.25)), 0).UTC()),
		fmt.Sprintf("%v", time.Unix(int64(s.qt.Query(0.5)), 0).UTC()),
		fmt.Sprintf("%v", time.Unix(int64(s.qt.Query(0.75)), 0).UTC()),
		fmt.Sprintf("%v", time.Unix(int64(s.qt.Query(0.90)), 0).UTC()),
		fmt.Sprintf("%v", time.Unix(int64(s.qt.Query(0.95)), 0).UTC()),
		fmt.Sprintf("%v", time.Unix(int64(s.qt.Query(0.99)), 0).UTC()),
		fmt.Sprintf("%v", time.Unix(s.last, 0).UTC()),
	}); err != nil {
		return err
	}

	csvw.Flush()
	return csvw.Error()
}

// The Stats output gathers statistics about the sizes of log lines it sees.
type Stats struct {
	totaln uint64 // total processed lines
	errn   uint64 // number of lines that were skipped because of errors

	cfg     baker.OutputParams
	csvPath string
	sizes   sizeStats      // log line sizes stats
	fields  []fieldStats   // per-field stats
	times   timestampStats // timestamps stats
}

// NewStats returns a new Stats Baker output.
func NewStats(cfg baker.OutputParams) (baker.Output, error) {
	dcfg := cfg.DecodedConfig.(*StatsConfig)
	dcfg.fillDefaults()

	// Ensure output file is writable
	outf, err := os.Create(dcfg.CSVPath)
	if err != nil {
		return nil, fmt.Errorf("can't create %s: %v", dcfg.CSVPath, err)
	}
	outf.Close()

	fstats := make([]fieldStats, 0)
	for _, field := range cfg.Fields {
		fstats = append(fstats, fieldStats{
			m:       make(map[string]uint),
			field:   field,
			empties: dcfg.CountEmptyFields,
		})
	}

	var idx baker.FieldIndex = -1
	if dcfg.TimestampField != "" {
		var ok bool
		idx, ok = cfg.FieldByName(dcfg.TimestampField)
		if !ok {
			return nil, fmt.Errorf("Cannot find field %s", dcfg.TimestampField)
		}
	}

	return &Stats{
		csvPath: dcfg.CSVPath,
		cfg:     cfg,
		sizes: sizeStats{
			smallest: math.MaxUint32,
			biggest:  0,
			qt:       quantile.NewTargeted(0.01, 0.05, 0.1, 0.25, 0.5, 0.75, 0.90, 0.95, 0.99),
		},
		fields: fstats,
		times: timestampStats{
			qt:       quantile.NewTargeted(0.01, 0.05, 0.1, 0.25, 0.5, 0.75, 0.90, 0.95, 0.99),
			first:    math.MaxInt64,
			last:     0,
			fieldIdx: idx,
		},
	}, nil
}

// Run implements baker.Output
func (s *Stats) Run(input <-chan baker.OutputRecord, _ chan<- string) error {
	ll := s.cfg.CreateRecord()

	valid := false
	for raw := range input {
		atomic.AddUint64(&s.totaln, 1)
		ll.Parse(raw.Record, nil)
		if valid, _ = s.cfg.ValidateRecord(ll); !valid {
			atomic.AddUint64(&s.errn, 1)
			continue
		}
		for i := range s.fields {
			s.fields[i].add(ll)
		}
		s.sizes.add(uint32(len(raw.Record)), ll, valid)
		if s.times.fieldIdx != -1 {
			s.times.add(ll)
		}
	}
	if err := s.createStatsCSV(); err != nil {
		return fmt.Errorf("can't open %s: %v", s.csvPath, err)
	}
	return nil
}

func (s *Stats) createStatsCSV() error {
	buf := &bytes.Buffer{}
	fmt.Fprintln(buf, "section,log line sizes,distribution of log lines sizes")
	(&s.sizes).print(buf)
	if s.times.fieldIdx != -1 {
		fmt.Fprintln(buf, "section,timestamps,distribution of timestamps")
		(&s.times).print(buf)
	}
	for i := range s.fields {
		fname := s.cfg.FieldNames[s.fields[i].field]
		fmt.Fprintf(buf, "section,%s,distribution of number of log lines per distinct %s value\n", fname, fname)
		s.fields[i].print(buf, s.cfg.FieldNames)
	}
	return os.WriteFile(s.csvPath, buf.Bytes(), os.ModePerm)

}

// Stats implements baker.Output
func (s *Stats) Stats() baker.OutputStats {
	return baker.OutputStats{
		NumProcessedLines: int64(atomic.LoadUint64(&s.totaln)),
		NumErrorLines:     int64(atomic.LoadUint64(&s.errn)),
	}
}

// CanShard implements baker.Output
func (s *Stats) CanShard() bool { return true }
