package log

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"testing"
)

var defaultLogger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))

func Default() *slog.Logger { return defaultLogger }

func SetDefault(l *slog.Logger) { defaultLogger = l }

func Printf(format string, v ...any) {
	defaultLogger.Info("applecontainer", "msg", fmt.Sprintf(format, v...))
}

func TestLogger(tb testing.TB) *slog.Logger {
	tb.Helper()
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
