package log

import (
	"bytes"
	"log/slog"
	"testing"
)

func TestTestLoggerIsNotNil(t *testing.T) {
	l := TestLogger(t)
	if l == nil {
		t.Fatal("TestLogger returned nil")
	}
}

func TestDefaultLoggerIsNotNil(t *testing.T) {
	l := Default()
	if l == nil {
		t.Fatal("Default returned nil")
	}
}

func TestSetDefaultRoutesPrintf(t *testing.T) {
	orig := Default()
	t.Cleanup(func() { SetDefault(orig) })

	var buf bytes.Buffer
	SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	Printf("captured: %s", "yes")

	if buf.Len() == 0 {
		t.Fatal("Printf did not route to the logger set via SetDefault")
	}
}
