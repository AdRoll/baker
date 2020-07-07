package inputtest

import (
	"encoding/hex"
	"math/rand"
	"strconv"
	"time"

	"github.com/AdRoll/baker"
)

// RandomDesc describes the Random input.
var RandomDesc = baker.InputDesc{
	Name:   "Random",
	New:    NewRandom,
	Config: &RandomConfig{},
}

// RandomConfig specifies the number of random log lines to generate as well as
// the seed for the PRNG.
type RandomConfig struct {
	NumLines int
	RandSeed int64
}

func (cfg *RandomConfig) fillDefaults() {
	if cfg.NumLines == 0 {
		cfg.NumLines = 1000
	}
}

// A Random input is baker input used for testing. It generates a specified
// number of random, but valid, log lines.
type Random struct {
	Cfg *RandomConfig
}

// NewRandom creates a Random baker input.
func NewRandom(cfg baker.InputParams) (baker.Input, error) {
	if cfg.DecodedConfig == nil {
		cfg.DecodedConfig = &RandomConfig{}
	}
	dcfg := cfg.DecodedConfig.(*RandomConfig)
	dcfg.fillDefaults()
	return &Random{
		Cfg: dcfg,
	}, nil
}

func (r *Random) Run(output chan<- *baker.Data) error {

	rand := rand.New(rand.NewSource(r.Cfg.RandSeed))

	tlen := 15 * 24 * time.Hour
	types := []string{"t1", "t2", "t3", "t4"}

	var buf []byte
	for i := 0; i < r.Cfg.NumLines; i++ {
		var ll baker.LogLine

		var t1 [16]byte
		rand.Read(t1[:])
		ll.Set(0, []byte(hex.EncodeToString(t1[:])))

		t2 := time.Date(2015, 8, 1, 15, 0, 0, 0, time.UTC).Add(time.Duration(rand.Int63n(int64(tlen))))
		ll.Set(1, []byte(strconv.Itoa(int(t2.Unix()))))

		t3 := types[rand.Intn(len(types))]
		ll.Set(2, []byte(t3))

		buf = ll.ToText(buf)
		buf = append(buf, '\n')
		if i%16 == 0 || i == r.Cfg.NumLines-1 {
			output <- &baker.Data{Bytes: buf}
			buf = nil
		}
	}
	return nil
}

func (r *Random) Stop()                           {}
func (r *Random) FreeMem(data *baker.Data)        {}
func (r *Random) Stats() (stats baker.InputStats) { return }
