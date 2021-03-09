package output

import (
	"net/http"
	"sync/atomic"

	"github.com/AdRoll/baker"
	"github.com/AdRoll/baker/output/websocket"
	log "github.com/sirupsen/logrus"
)

var WebSocketDesc = baker.OutputDesc{
	Name:   "WebSocket",
	New:    NewWebSocket,
	Config: &WebSocketConfig{},
	Raw:    false,
	Help:   "This output writes the filtered log lines into any connected WebSocket client.\n",
}

type WebSocketConfig struct{}

func (cfg *WebSocketConfig) fillDefaults() {}

type WebSocket struct {
	Cfg *WebSocketConfig

	Fields []baker.FieldIndex

	fieldByName func(string) (baker.FieldIndex, bool)
	totaln      int64
}

func NewWebSocket(cfg baker.OutputParams) (baker.Output, error) {
	log.WithFields(log.Fields{"fn": "NewWebSocket", "idx": cfg.Index}).Info("Initializing")

	dcfg := cfg.DecodedConfig.(*WebSocketConfig)
	dcfg.fillDefaults()

	return &WebSocket{
		Cfg:         dcfg,
		Fields:      cfg.Fields,
		fieldByName: cfg.FieldByName,
	}, nil
}

// websocket server

func (w *WebSocket) Run(input <-chan baker.OutputRecord, _ chan<- string) error {
	cfg := websocket.Conf{
		Fields:      w.Fields,
		FieldByName: w.fieldByName,
	}
	server := websocket.NewServer(cfg)
	go server.Listen()

	go func() {
		log.Fatal(http.ListenAndServe(":8080", nil))
	}()

	log.Info("WS Ready to receive records")
	for lldata := range input {
		server.SendAll(lldata.Fields)
		atomic.AddInt64(&w.totaln, int64(1))
	}

	return nil
}

func (w *WebSocket) Stats() baker.OutputStats {
	return baker.OutputStats{
		NumProcessedLines: atomic.LoadInt64(&w.totaln),
	}
}

func (b *WebSocket) CanShard() bool {
	return false
}
