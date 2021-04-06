package filter

import (
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/AdRoll/baker"
	"github.com/fernet/fernet-go"
	log "github.com/sirupsen/logrus"
)

const cryptHelp = `
This filter encrypts or decrypts a field and writes the resulting value to another (or the same) field.

Supported algorithms:
 - fernet

### Fernet configuration

 - **Key**: 256-bit key used to encrypt/decrypt the token.
 - **TTL**: optional duration (in seconds). When set, the key must have been signed at most TTL ago, or decryption will fail. Only applicable for decryption.
`

var CryptDesc = baker.FilterDesc{
	Name:   "Crypt",
	New:    NewCrypt,
	Config: &CryptConfig{},
	Help:   cryptHelp,
}

type CryptConfig struct {
	Algorithm       string            `help:"Name of the algorithm to use for crypting/decrypting" required:"true"`
	Decrypt         bool              `help:"True for decrypting, false for encrypting" default:"false"`
	SrcField        string            `help:"Name of the field to crypt/decrypt" required:"true"`
	DstField        string            `help:"Name of the field to write the result to" required:"true"`
	AlgorithmConfig map[string]string `help:"AlgorithmConf contains configurations required by the chosen algorithm"`
}

type cryptAlgorithm interface {
	parseConf(map[string]string) error
	encrypt([]byte) ([]byte, error)
	decrypt([]byte) ([]byte, error)
}

type Crypt struct {
	src         baker.FieldIndex
	dst         baker.FieldIndex
	algorithm   cryptAlgorithm
	transformFn func([]byte) ([]byte, error)

	// Shared state
	numFilteredLines int64
}

func NewCrypt(cfg baker.FilterParams) (baker.Filter, error) {
	dcfg := cfg.DecodedConfig.(*CryptConfig)

	src, ok := cfg.FieldByName(dcfg.SrcField)
	if !ok {
		return nil, fmt.Errorf("unknown field %q", dcfg.SrcField)
	}

	dst, ok := cfg.FieldByName(dcfg.DstField)
	if !ok {
		return nil, fmt.Errorf("unknown field %q", dcfg.DstField)
	}

	f := &Crypt{src: src, dst: dst}

	switch dcfg.Algorithm {
	case "fernet":
		f.algorithm = &cryptFernet{}
	default:
		return nil, fmt.Errorf("unsupported algorithm %s", dcfg.Algorithm)
	}
	if err := f.algorithm.parseConf(dcfg.AlgorithmConfig); err != nil {
		return nil, fmt.Errorf("invalid %s algorithm configuration: %s", dcfg.Algorithm, err)
	}

	if dcfg.Decrypt {
		f.transformFn = f.algorithm.decrypt
	} else {
		f.transformFn = f.algorithm.encrypt
	}
	return f, nil
}

func (f *Crypt) Stats() baker.FilterStats {
	return baker.FilterStats{
		NumFilteredLines: atomic.LoadInt64(&f.numFilteredLines),
	}
}

func (f *Crypt) Process(r baker.Record, next func(baker.Record)) {
	newVal, err := f.transformFn(r.Get(f.src))
	if err != nil {
		log.Errorf("can't process line: %v", err)
		atomic.AddInt64(&f.numFilteredLines, 1)
		return
	}

	r.Set(f.dst, newVal)
	next(r)
}

type cryptFernet struct {
	key *fernet.Key
	ttl time.Duration
}

func (alg *cryptFernet) parseConf(conf map[string]string) error {
	keyStr, ok := conf["Key"]
	if !ok {
		return fmt.Errorf(`can't find "Key" conf for Fernet algorithm`)
	}

	key, err := fernet.DecodeKey(keyStr)
	if err != nil {
		return err
	}
	alg.key = key

	ttl := 0
	ttlStr, ok := conf["TTL"]
	if ok {
		ttl, err = strconv.Atoi(ttlStr)
		if err != nil {
			return fmt.Errorf(`can't parse "TTL" as seconds: %v`, err)
		}
	}
	alg.ttl = time.Duration(ttl) * time.Second

	return err
}

func (alg *cryptFernet) encrypt(msg []byte) ([]byte, error) {
	return fernet.EncryptAndSign(msg, alg.key)
}

func (alg *cryptFernet) decrypt(crypted []byte) ([]byte, error) {
	decrypted := fernet.VerifyAndDecrypt(crypted, alg.ttl, []*fernet.Key{alg.key})
	if decrypted == nil {
		return nil, fmt.Errorf("can't decrypt the field value")
	}
	return decrypted, nil
}
