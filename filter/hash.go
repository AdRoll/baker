package filter

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync/atomic"

	"github.com/AdRoll/baker"
	log "github.com/sirupsen/logrus"
)

// HashDesc describes the Hash filter
var HashDesc = baker.FilterDesc{
	Name:   "Hash",
	New:    NewHash,
	Config: &HashConfig{},
	Help: `This filter hashes a field using a specified hash function and writes the value 
to another (or the same) field. In order to have control over the set of characters
present, the hashed value can optionally be encoded.
	
	
Supported hash functions:
 - md5
 - sha256

Supported encodings:
- hex (hexadecimal encoding)
`,
}

type HashConfig struct {
	SrcField string `help:"Name of the field to hash" required:"true"`
	DstField string `help:"Name of the field to write the result to" required:"true"`
	Function string `help:"Name of the hash function to use" required:"true"`
	Encoding string `help:"Name of the encoding function to use" required:"false"`
}

type Hash struct {
	numProcessedLines int64
	numFilteredLines  int64

	src    baker.FieldIndex
	dst    baker.FieldIndex
	hash   func([]byte) ([]byte, error)
	encode func([]byte) ([]byte, error)
}

func NewHash(cfg baker.FilterParams) (baker.Filter, error) {
	if cfg.DecodedConfig == nil {
		cfg.DecodedConfig = &HashConfig{}
	}
	dcfg := cfg.DecodedConfig.(*HashConfig)

	src, ok := cfg.FieldByName(dcfg.SrcField)
	if !ok {
		return nil, fmt.Errorf("can't find the SrcField %s", dcfg.SrcField)
	}

	dst, ok := cfg.FieldByName(dcfg.DstField)
	if !ok {
		return nil, fmt.Errorf("can't find the DstField %s", dcfg.DstField)
	}

	h := &Hash{
		src: src,
		dst: dst,
	}

	switch dcfg.Function {
	case "md5":
		h.hash = func(b []byte) ([]byte, error) {
			sum := md5.Sum(b)
			return sum[:], nil
		}
	case "sha256":
		h.hash = func(b []byte) ([]byte, error) {
			sum := sha256.Sum256(b)
			return sum[:], nil
		}
	default:
		return nil, fmt.Errorf("unsupported hash function %s", dcfg.Function)
	}

	switch dcfg.Encoding {
	case "hex":
		h.encode = func(b []byte) ([]byte, error) {
			dst := make([]byte, hex.EncodedLen(len(b)))
			hex.Encode(dst, b)
			return dst, nil
		}
	case "": // pass through
		h.encode = func(b []byte) ([]byte, error) { return b, nil }
	default:
		return nil, fmt.Errorf("unsupported encoding %s", dcfg.Encoding)
	}

	return h, nil
}

func (h *Hash) Stats() baker.FilterStats {
	return baker.FilterStats{
		NumProcessedLines: atomic.LoadInt64(&h.numProcessedLines),
		NumFilteredLines:  atomic.LoadInt64(&h.numFilteredLines),
	}
}

func (h *Hash) Process(r baker.Record, next func(baker.Record)) {
	atomic.AddInt64(&h.numProcessedLines, 1)

	hashed, err := h.hash(r.Get(h.src))
	if err != nil {
		log.Errorf("can't process record, hashing failed: %v", err)
		atomic.AddInt64(&h.numFilteredLines, 1)
		return
	}

	encoded, err := h.encode(hashed)
	if err != nil {
		log.Errorf("can't process record, encoding failed: %v", err)
		atomic.AddInt64(&h.numFilteredLines, 1)
		return
	}

	r.Set(h.dst, encoded)
	next(r)
}
