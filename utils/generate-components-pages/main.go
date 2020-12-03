package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/AdRoll/baker"
	"github.com/AdRoll/baker/filter"
	"github.com/AdRoll/baker/input"
	"github.com/AdRoll/baker/output"
	"github.com/AdRoll/baker/upload"
)

var baseFolder = ""

var components = baker.Components{
	Inputs:  input.All,
	Filters: filter.All,
	Outputs: output.All,
	Uploads: upload.All,
}

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s <destination folder>\n", os.Args[0])
		os.Exit(1)
	}

	fileInfo, err := os.Stat(os.Args[1])
	if err != nil {
		panic(err)
	}
	if !fileInfo.IsDir() {
		panic("the provided arguments isn't a folder")
	}

	count := 1

	// Inputs
	dest := filepath.Join(os.Args[1], "Inputs")
	if err := prepareFolder(dest, count); err != nil {
		panic(err)
	}
	count++

	for _, comps := range components.Inputs {
		fname := filepath.Join(dest, fmt.Sprintf("%s.md", comps.Name))

		if err := writeCompFile(fname, comps.Name, count); err != nil {
			panic(err)
		}

		count++
	}

	// Filters
	dest = filepath.Join(os.Args[1], "Filters")
	if err := prepareFolder(dest, count); err != nil {
		panic(err)
	}
	count++

	for _, comps := range components.Filters {
		fname := filepath.Join(dest, fmt.Sprintf("%s.md", comps.Name))

		if err := writeCompFile(fname, comps.Name, count); err != nil {
			panic(err)
		}

		count++
	}

	// Outputs
	dest = filepath.Join(os.Args[1], "Outputs")
	if err := prepareFolder(dest, count); err != nil {
		panic(err)
	}
	count++

	for _, comps := range components.Outputs {
		fname := filepath.Join(dest, fmt.Sprintf("%s.md", comps.Name))

		if err := writeCompFile(fname, comps.Name, count); err != nil {
			panic(err)
		}

		count++
	}

	// Uploads
	dest = filepath.Join(os.Args[1], "Uploads")
	if err := prepareFolder(dest, count); err != nil {
		panic(err)
	}
	count++

	for _, comps := range components.Uploads {
		fname := filepath.Join(dest, fmt.Sprintf("%s.md", comps.Name))

		if err := writeCompFile(fname, comps.Name, count); err != nil {
			panic(err)
		}

		count++
	}
}

func prepareFolder(dest string, count int) error {
	if err := os.MkdirAll(dest, os.ModePerm); err != nil {
		return err
	}

	return writeIndexFile(dest, count)
}

func writeIndexFile(dest string, count int) error {
	f, err := os.Create(filepath.Join(dest, "_index.md"))
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	defer w.Flush()

	return writeMarkdownHeader(w, filepath.Base(dest), count)
}

func writeCompFile(dest, comp string, count int) error {
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	defer w.Flush()

	if err := writeMarkdownHeader(w, comp, count); err != nil {
		return err
	}

	if err := writeAPILinks(w, dest); err != nil {
		return err
	}

	return baker.PrintHelp(w, comp, components, baker.HelpFormatMarkdown)

}

func writeMarkdownHeader(w *bufio.Writer, title string, count int) error {
	d := time.Now().Format("2006-01-02")

	s := fmt.Sprintf("---\ntitle: \"%s\"\nweight: %d\ndate: %s\n---\n", title, count, d)

	_, err := w.WriteString(s)
	return err
}

func writeAPILinks(w *bufio.Writer, dest string) error {
	var c string
	switch {
	case strings.Contains(dest, "Inputs/"):
		c = "input"
	case strings.Contains(dest, "Filters/"):
		c = "filter"
	case strings.Contains(dest, "Outputs/"):
		c = "output"
	case strings.Contains(dest, "Uploads/"):
		c = "upload"
	default:
		return fmt.Errorf("unexpected component path %s", dest)
	}

	s := fmt.Sprintf("{{%% pageinfo color=\"primary\" %%}}")
	s = fmt.Sprintf("%s\n\n**Read the [API documentation &raquo;](https://pkg.go.dev/github.com/AdRoll/baker/%s)**\n", s, c)
	s = fmt.Sprintf("%s{{%% /pageinfo %%}}\n\n", s)

	_, err := w.WriteString(s)
	return err
}
