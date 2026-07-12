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
	defaultLogger.Info(fmt.Sprintf(format, v...)) // Or just standard log if preferred, but it keeps compat
}

func TestLogger(tb testing.TB) *slog.Logger {
	tb.Helper()
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
