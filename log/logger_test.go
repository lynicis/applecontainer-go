package log

import (
	"fmt"
	"testing"
)

func TestTestLoggerWritesViaTLogf(t *testing.T) {
	l := TestLogger(t)
	if l == nil {
		t.Fatal("TestLogger returned nil")
	}
	// TestLogger.Printf routes through t.Logf; this line appears in -v output.
	l.Printf("applecontainer log line: %s=%d", "port", 5432)
}

func TestDefaultLoggerIsNotNil(t *testing.T) {
	l := Default()
	if l == nil {
		t.Fatal("Default returned nil")
	}
	// Must not panic.
	l.Printf("no-op ok: %v", 42)
}

func TestSetDefaultRoutesPrintf(t *testing.T) {
	orig := Default()
	t.Cleanup(func() { SetDefault(orig) })

	rec := &recordLogger{}
	SetDefault(rec)
	Printf("captured: %s", "yes")

	if len(rec.lines) == 0 {
		t.Fatal("Printf did not route to the logger set via SetDefault")
	}
	if rec.lines[0] != "captured: yes" {
		t.Fatalf("got %q want %q", rec.lines[0], "captured: yes")
	}
}

type recordLogger struct{ lines []string }

func (r *recordLogger) Printf(format string, v ...any) {
	r.lines = append(r.lines, fmt.Sprintf(format, v...))
}
