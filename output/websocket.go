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
	New:    NewWebSocketWriter,
	Config: &WebSocketWriterConfig{},
	Raw:    false,
	Help:   "This output writes the filtered log lines into any conenct WebSocket client.\n",
}

type WebSocketWriterConfig struct{}

func (cfg *WebSocketWriterConfig) fillDefaults() {}

type WebSocketWriter struct {
	Cfg *WebSocketWriterConfig

	Fields []baker.FieldIndex

	fieldByName func(string) (baker.FieldIndex, bool)
	totaln      int64
}

func NewWebSocketWriter(cfg baker.OutputParams) (baker.Output, error) {
	log.WithFields(log.Fields{"fn": "NewWebSocketWriter", "idx": cfg.Index}).Info("Initializing")

	if cfg.DecodedConfig == nil {
		cfg.DecodedConfig = &WebSocketWriterConfig{}
	}
	dcfg := cfg.DecodedConfig.(*WebSocketWriterConfig)
	dcfg.fillDefaults()

	return &WebSocketWriter{
		Cfg:         dcfg,
		Fields:      cfg.Fields,
		fieldByName: cfg.FieldByName,
	}, nil
}

// websocket server

func (w *WebSocketWriter) Run(input <-chan baker.OutputRecord, _ chan<- string) error {
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

func (w *WebSocketWriter) Stats() baker.OutputStats {
	return baker.OutputStats{
		NumProcessedLines: atomic.LoadInt64(&w.totaln),
	}
}

func (b *WebSocketWriter) CanShard() bool {
	return false
}
