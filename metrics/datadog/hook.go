package datadog

import (
	"fmt"
	"strings"

	"github.com/DataDog/datadog-go/statsd"
	log "github.com/sirupsen/logrus"
)

type hook struct {
	levels []log.Level
	client *statsd.Client
	tags   []string
	host   string
}

// NewHook returns a Logrus hook that forwards log entries as events to a
// statsd client, such as the datadog-agent.
//
// Log entries with a level higher than level are discarded.
// host is used to fill the Hostname field of statsd events, its purpose it NOT
// to serve as configuring the stats connection (the client must already be
// configured).
// tags is a list of tags to include with all events.
//
// TODO[aurelien]: since we directly pass the client, we shouldn't need to pass
// the tags as the client as a Tags fields for global tags set on each metrics.
// However, the metrics package in Baker doesn't use client.Tags and pass the
// tags each time, so there's a bit of refactoring to do.
func NewHook(level log.Level, client *statsd.Client, host string, tags []string) log.Hook {
	levels := make([]log.Level, level+1)
	copy(levels[:level+1], log.AllLevels)

	return &hook{
		client: client,
		levels: levels,
		tags:   tags,
		host:   host,
	}
}

func (h *hook) Levels() []log.Level {
	return h.levels
}

func (h *hook) Fire(ent *log.Entry) error {
	// Format the statsd event message as message + fields as k=v:
	// example "this is message k1=v1 k2=v2 k3=v3""
	buf := strings.Builder{}
	buf.WriteString(ent.Message)
	for k, v := range ent.Data {
		buf.WriteByte(' ')
		buf.WriteString(k)
		buf.WriteByte('=')
		fmt.Fprintf(&buf, "%v", v)
	}

	evt := &statsd.Event{
		Tags:           h.tags,
		Timestamp:      ent.Time,
		SourceTypeName: "baker",
		AlertType:      levelToAlertType(ent.Level),
		Text:           "event text",
		Title:          buf.String(),
		Hostname:       h.host,
	}
	h.client.Event(evt)
	return nil
}

func levelToAlertType(level log.Level) statsd.EventAlertType {
	switch level {
	case log.PanicLevel:
	case log.FatalLevel:
	case log.ErrorLevel:
		return statsd.Error
	case log.WarnLevel:
		return statsd.Warning
	case log.InfoLevel:
	case log.DebugLevel:
	case log.TraceLevel:
		return statsd.Info
	}
	return statsd.Info
}
