package baker

import (
	"testing"
)

func TestFillCreateRecordDefault(t *testing.T) {
	t.Run("without separator in conf", func(t *testing.T) {
		cfg := Config{}
		if err := cfg.fillCreateRecordDefault(); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		ll := cfg.createRecord().(*LogLine)
		if ll.FieldSeparator != DefaultLogLineFieldSeparator {
			t.Fatalf("want: %v, got: %v", DefaultLogLineFieldSeparator, ll.FieldSeparator)
		}
	})

	t.Run("with comma separator", func(t *testing.T) {
		cfg := Config{
			CSV: ConfigCSV{
				FieldSeparator: ",",
			},
		}
		if err := cfg.fillCreateRecordDefault(); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		ll := cfg.createRecord().(*LogLine)
		if ll.FieldSeparator != DefaultLogLineFieldSeparator {
			t.Fatalf("want: %v, got: %v", DefaultLogLineFieldSeparator, ll.FieldSeparator)
		}
	})

	t.Run("with record separator", func(t *testing.T) {
		cfg := Config{
			CSV: ConfigCSV{
				FieldSeparator: "\u001e",
			},
		}
		if err := cfg.fillCreateRecordDefault(); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		ll := cfg.createRecord().(*LogLine)
		if ll.FieldSeparator != 30 {
			t.Fatalf("want: %v, got: %v", 30, ll.FieldSeparator)
		}
	})

	t.Run("with dot separator", func(t *testing.T) {
		cfg := Config{
			CSV: ConfigCSV{
				FieldSeparator: ".",
			},
		}
		if err := cfg.fillCreateRecordDefault(); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		ll := cfg.createRecord().(*LogLine)
		if ll.FieldSeparator != 46 {
			t.Fatalf("want: %v, got: %v", 46, ll.FieldSeparator)
		}
	})

	t.Run("wrong ascii separator", func(t *testing.T) {
		cfg := Config{
			CSV: ConfigCSV{
				FieldSeparator: "è",
			},
		}
		if err := cfg.fillCreateRecordDefault(); err == nil {
			t.Fatal("unexpected nil err")
		}
	})

	t.Run("separator too long", func(t *testing.T) {
		cfg := Config{
			CSV: ConfigCSV{
				FieldSeparator: ".,",
			},
		}
		if err := cfg.fillCreateRecordDefault(); err == nil {
			t.Fatal("unexpected nil err")
		}
	})
}
