package datadog

import (
	"net"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestClientMetrics(t *testing.T) {
	conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("can't listen on udp: %v", err)
	}
	defer conn.Close()

	t.Run("metrics", func(t *testing.T) {
		quit := make(chan struct{})
		done := make(chan struct{})

		var packets []string

		go func() {
			defer func() { close(done) }()

			const maxsize = 32 * 1024
			p := make([]byte, maxsize)
			for {
				select {
				case <-quit:
					return
				default:
					conn.SetDeadline(time.Now().Add(100 * time.Millisecond))
					n, _, err := conn.ReadFrom(p)
					if err != nil {
						break
					}

					packets = append(packets, string(p[:n]))
				}
			}
		}()

		cfg := &Config{
			Host:   conn.LocalAddr().String(),
			Prefix: "prefix.",
			Tags:   []string{"basetag1:abc", "basetag2:xyz"},
		}

		c, err := newClient(cfg)
		if err != nil {
			t.Fatalf("can't create datadog metrics client: %v", err)
		}

		c.DeltaCount("delta", 1)
		c.DeltaCountWithTags("delta-with-tags", 2, []string{"tag1:1", "tag2:2"})

		c.Duration("duration", 3*time.Millisecond)
		c.DurationWithTags("duration-with-tags", 4*time.Millisecond, []string{"tag2:2", "tag3:3"})

		c.Gauge("gauge", 5)
		c.GaugeWithTags("gauge-with-tags", 6, []string{"tag3:3", "tag4:4"})

		c.Histogram("histogram", 7)
		c.Histogram("histogram", 8)
		c.Histogram("histogram", 9)
		c.Histogram("histogram", 10)
		c.Histogram("histogram", 11)

		c.HistogramWithTags("histogram-with-tags", 12, []string{"tag4:4", "tag5:5"})
		c.HistogramWithTags("histogram-with-tags", 13, []string{"tag4:4", "tag5:5"})
		c.HistogramWithTags("histogram-with-tags", 14, []string{"tag4:4", "tag5:5"})
		c.HistogramWithTags("histogram-with-tags", 15, []string{"tag4:4", "tag5:5"})
		c.HistogramWithTags("histogram-with-tags", 16, []string{"tag4:4", "tag5:5"})

		c.RawCount("raw-count", 17)
		c.RawCountWithTags("raw-count-with-tags", 18, []string{"tag5:5", "tag6:6"})

		logrus.WithFields(logrus.Fields{"field1": 27, "field2": "spiral"}).Warn("warn log message")

		time.Sleep(500 * time.Millisecond)
		close(quit)
		<-done

		want := []string{"prefix.delta:1|c|#basetag1:abc,basetag2:xyz",
			"prefix.delta-with-tags:2|c|#basetag1:abc,basetag2:xyz,tag1:1,tag2:2",
			"prefix.duration:3.000000|ms|#basetag1:abc,basetag2:xyz",
			"prefix.duration-with-tags:4.000000|ms|#basetag1:abc,basetag2:xyz,tag2:2,tag3:3",
			"prefix.gauge:5|g|#basetag1:abc,basetag2:xyz",
			"prefix.gauge-with-tags:6|g|#basetag1:abc,basetag2:xyz,tag3:3,tag4:4",
			"prefix.histogram:7|h|#basetag1:abc,basetag2:xyz",
			"prefix.histogram:8|h|#basetag1:abc,basetag2:xyz",
			"prefix.histogram:9|h|#basetag1:abc,basetag2:xyz",
			"prefix.histogram:10|h|#basetag1:abc,basetag2:xyz",
			"prefix.histogram:11|h|#basetag1:abc,basetag2:xyz",
			"prefix.histogram-with-tags:12|h|#basetag1:abc,basetag2:xyz,tag4:4,tag5:5",
			"prefix.histogram-with-tags:13|h|#basetag1:abc,basetag2:xyz,tag4:4,tag5:5",
			"prefix.histogram-with-tags:14|h|#basetag1:abc,basetag2:xyz,tag4:4,tag5:5",
			"prefix.histogram-with-tags:15|h|#basetag1:abc,basetag2:xyz,tag4:4,tag5:5",
			"prefix.histogram-with-tags:16|h|#basetag1:abc,basetag2:xyz,tag4:4,tag5:5",
			"prefix.raw-count:17|c|#basetag1:abc,basetag2:xyz",
			"prefix.raw-count-with-tags:18|c|#basetag1:abc,basetag2:xyz,tag5:5,tag6:6",
		}

		// It's unlikely, but still possible, that we received multiple packets
		// But we anyway want to split the packets on '\n' and remove them.
		var got []string
		for _, p := range packets {
			for _, s := range strings.Split(p, "\n") {
				// Look for the tag section and order tags
				pos := strings.Index(s, "|#")
				if pos == -1 {
					t.Errorf("didn't find tag section in %q", s)
				}
				tags := strings.Split(s[pos+2:], ",")
				sort.Strings(tags)

				m := s[:pos] + "|#" + strings.Join(tags, ",")
				got = append(got, m)
			}
		}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("received %v, want %v", packets, want)
		}
	})

	t.Run("logs", func(t *testing.T) {
		quit := make(chan struct{})
		done := make(chan struct{})
		packet := ""

		go func() {
			defer func() { close(done) }()

			const maxsize = 32 * 1024
			p := make([]byte, maxsize)
			for {
				select {
				case <-quit:
					return
				default:
					conn.SetDeadline(time.Now().Add(100 * time.Millisecond))
					n, _, err := conn.ReadFrom(p)
					if err != nil {
						break
					}

					packet += string(p[:n])
				}
			}
		}()

		cfg := &Config{
			Host:     conn.LocalAddr().String(),
			Prefix:   "prefix.",
			Tags:     []string{"basetag1:abc", "basetag2:xyz"},
			SendLogs: true,
		}

		_, err := newClient(cfg)
		if err != nil {
			t.Fatalf("can't create datadog metrics client: %v", err)
		}

		logrus.WithFields(logrus.Fields{"field1": 27, "field2": "spiral"}).Warn("warn log message")

		time.Sleep(500 * time.Millisecond)
		close(quit)
		<-done

		// Exact statds events depends on the timestamp and the order of map
		// iteration let's just look at the presence of some values and consider
		// the log event sent/received, we're not testing statd events format
		// specification anyway.
		want := []string{
			"warn log message",
			"field1=27",
			"field2=spiral",
			"#basetag1:abc,basetag2:xyz",
			"warning",
			"s:baker",
			"event text",
		}
		for _, w := range want {
			if !strings.Contains(packet, w) {
				t.Errorf("want event to contain %q but didn't\n packet = %q", w, packet)
			}
		}
	})
}
