package output

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AdRoll/baker"
	"github.com/AdRoll/baker/input/inputtest"
	"github.com/AdRoll/baker/testutil"
)

func BenchmarkFilesOutput(b *testing.B) {
	defer testutil.DisableLogging()()
	dir, rmdir := testutil.TempDir(b)
	defer rmdir()

	toml := fmt.Sprintf(`
	[input]
	name="Random"
		[input.config]
		numlines = 10000

	[output]
	name="Files"
	procs=1

		[output.config]
		pathstring = "%s"
		zstdcompressionlevel = 0
	`, filepath.Join(dir, "log.zst"))

	c := baker.Components{
		Inputs:  []baker.InputDesc{inputtest.RandomDesc},
		Outputs: []baker.OutputDesc{FileWriterDesc},
	}

	cfg, err := baker.NewConfigFromToml(strings.NewReader(toml), c)
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		b.StopTimer()
		topology, err := baker.NewTopologyFromConfig(cfg)
		if err != nil {
			b.Fatal(err)
		}
		b.StartTimer()
		topology.Start()
		topology.Wait()
		topology.Stop()
	}
}
