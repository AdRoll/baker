package datadog

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/AdRoll/baker/testutil"
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
		if err := c.Close(); err != nil {
			t.Fatalf("close error: %v", err)
		}
		time.Sleep(500 * time.Millisecond)
		close(quit)
		<-done

		// It's unlikely, but still possible, that we received multiple packets,
		// in which case we need to split them.
		var re = regexp.MustCompile(`\|c:[[:xdigit:]]{8}-[[:xdigit:]]{4}-[[:xdigit:]]{4}-[[:xdigit:]]{4}-[[:xdigit:]]{12}`)
		var got []string
		for _, p := range packets {
			for _, s := range strings.Split(p, "\n") {
				if strings.TrimSpace(s) == "" {
					continue
				}
				// Remove the tag id added by datadog client on each tag.
				s := re.ReplaceAllString(s, "")
				// Look for the tag section and order tags so we get a deterministic output
				pos := strings.Index(s, "|#")
				tags := strings.Split(s[pos+2:], ",")
				sort.Strings(tags)
				got = append(got, s[:pos]+"|#"+strings.Join(tags, ","))
			}
		}
		sort.Strings(got)

		buf := bytes.Buffer{}
		for _, l := range got {
			fmt.Fprintln(&buf, l)
		}

		golden := filepath.Join("testdata", "TestClientMetrics.metrics.golden")
		if *testutil.UpdateGolden {
			os.WriteFile(golden, buf.Bytes(), os.ModePerm)
			t.Logf("updated: %q", golden)
		}
		testutil.DiffWithGolden(t, buf.Bytes(), golden)
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

		c, err := newClient(cfg)
		if err != nil {
			t.Fatalf("can't create datadog metrics client: %v", err)
		}

		logrus.WithFields(logrus.Fields{"field1": 27, "field2": "spiral"}).Warn("warn log message")
		if err := c.Close(); err != nil {
			t.Fatalf("close error: %v", err)
		}

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
