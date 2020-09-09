package datadog

import (
	"fmt"
	"reflect"
	"testing"

	log "github.com/sirupsen/logrus"
)

func TestHookLevels(t *testing.T) {
	tests := []struct {
		level log.Level
		want  []log.Level
	}{
		{level: log.PanicLevel, want: []log.Level{log.PanicLevel}},
		{level: log.InfoLevel, want: []log.Level{log.PanicLevel, log.FatalLevel, log.ErrorLevel, log.WarnLevel, log.InfoLevel}},
		{level: log.TraceLevel, want: log.AllLevels},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("level=%s", tt.level), func(t *testing.T) {
			h := NewHook(tt.level, nil, "", nil)
			if got := h.Levels(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("hook.Levels() = %v, want %v", got, tt.want)
			}
		})
	}
}
