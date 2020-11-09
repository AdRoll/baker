package baker

import (
	"fmt"
	"io"
	"reflect"
)

// GenerateMarkdownHelp generates markdown-formatted textual help for a Baker
// component from its description structure. Markdown is written into w.
func GenerateMarkdownHelp(w io.Writer, desc interface{}) error {
	if desc == nil {
		return fmt.Errorf("can't generate markdown help for a nil interface")
	}

	if reflect.TypeOf(desc).Kind() == reflect.Ptr {
		// dereference pointer
		desc = reflect.ValueOf(desc).Elem().Interface()
	}

	switch d := desc.(type) {
	case InputDesc:
		return genInputMarkdown(w, d)
	case FilterDesc:
		return genFilterMarkdown(w, d)
	case OutputDesc:
		return genOutputMarkdown(w, d)
	case UploadDesc:
		return genUploadMarkdown(w, d)
	case MetricsDesc:
		return genMetricsMarkdown(w, d)
	}

	return fmt.Errorf("can't generate markdown, unsupported type %T", desc)
}

func genInputMarkdown(w io.Writer, desc InputDesc) error {
	return nil
}

func genFilterMarkdown(w io.Writer, desc FilterDesc) error {
	return nil
}

func genOutputMarkdown(w io.Writer, desc OutputDesc) error {
	return nil
}

func genUploadMarkdown(w io.Writer, desc UploadDesc) error {
	return nil
}

func genMetricsMarkdown(w io.Writer, desc MetricsDesc) error {
	return nil
}
