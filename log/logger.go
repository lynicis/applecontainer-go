// Package log provides a minimal logger interface for applecontainer-go,
// ported from testcontainers-go's log package.
package log

import (
	stdlog "log"
	"os"
	"strings"
	"testing"
)

var (
	_ Logger = (*stdlog.Logger)(nil)
	_ Logger = (*noopLogger)(nil)
	_ Logger = (*testLogger)(nil)
)

// Logger defines the Logger interface.
type Logger interface {
	Printf(format string, v ...any)
}

// defaultLogger is the default Logger instance.
var defaultLogger Logger = &noopLogger{}

func init() {
	// Enable the default logger in testing with a verbose flag.
	if testing.Testing() {
		// Parse manually because testing.Verbose() panics unless flag.Parse() has run.
		for _, arg := range os.Args {
			if strings.EqualFold(arg, "-test.v=true") || strings.EqualFold(arg, "-v") {
				defaultLogger = stdlog.New(os.Stderr, "", stdlog.LstdFlags)
			}
		}
	}
}

// Default returns the default Logger instance.
func Default() Logger {
	return defaultLogger
}

// SetDefault sets the default Logger instance.
func SetDefault(logger Logger) {
	defaultLogger = logger
}

// Printf logs a formatted line via the default Logger.
func Printf(format string, v ...any) {
	defaultLogger.Printf(format, v...)
}

type noopLogger struct{}

// Printf implements Logger.
func (n noopLogger) Printf(_ string, _ ...any) {}

// TestLogger returns a Logger that writes through testing.TB.Logf, so library
// logs appear in the test output of the owning test.
func TestLogger(tb testing.TB) Logger {
	tb.Helper()
	return testLogger{TB: tb}
}

type testLogger struct {
	testing.TB
}

// Printf implements Logger.
func (t testLogger) Printf(format string, v ...any) {
	t.Helper()
	t.Logf(format, v...)
}
