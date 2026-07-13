package wait

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeTarget struct {
	hostFunc       func(ctx context.Context) (string, error)
	mappedPortFunc func(ctx context.Context, port string) (int, error)
	logsFunc       func(ctx context.Context) (io.ReadCloser, error)
	execFunc       func(ctx context.Context, cmd []string, opts ...any) (int, []byte, error)
	stateFunc      func(ctx context.Context) (string, int, error)
	statusFunc     func(ctx context.Context) (string, error)
	exitCodeFunc   func(ctx context.Context) (int, error)
	copyFileFunc   func(ctx context.Context, path string) (io.ReadCloser, error)
}

func (f *fakeTarget) Host(ctx context.Context) (string, error) {
	if f.hostFunc != nil {
		return f.hostFunc(ctx)
	}
	return "localhost", nil
}

func (f *fakeTarget) MappedPort(ctx context.Context, port string) (int, error) {
	if f.mappedPortFunc != nil {
		return f.mappedPortFunc(ctx, port)
	}
	return 0, nil
}

func (f *fakeTarget) Logs(ctx context.Context) (io.ReadCloser, error) {
	if f.logsFunc != nil {
		return f.logsFunc(ctx)
	}
	return io.NopCloser(bytes.NewReader(nil)), nil
}

func (f *fakeTarget) Exec(ctx context.Context, cmd []string, opts ...any) (int, []byte, error) {
	if f.execFunc != nil {
		return f.execFunc(ctx, cmd, opts...)
	}
	return 0, nil, nil
}

func (f *fakeTarget) State(ctx context.Context) (string, int, error) {
	if f.stateFunc != nil {
		return f.stateFunc(ctx)
	}
	status, err := f.StateStatus(ctx)
	if err != nil {
		return "", 0, err
	}
	code, err := f.StateExitCode(ctx)
	if err != nil {
		return "", 0, err
	}
	return status, code, nil
}

func (f *fakeTarget) StateStatus(ctx context.Context) (string, error) {
	if f.statusFunc != nil {
		return f.statusFunc(ctx)
	}
	return "running", nil
}

func (f *fakeTarget) StateExitCode(ctx context.Context) (int, error) {
	if f.exitCodeFunc != nil {
		return f.exitCodeFunc(ctx)
	}
	return 0, nil
}

func (f *fakeTarget) CopyFileFromContainer(ctx context.Context, path string) (io.ReadCloser, error) {
	if f.copyFileFunc != nil {
		return f.copyFileFunc(ctx, path)
	}
	return nil, errors.New("not found")
}

func TestForAllStrategy(t *testing.T) {
	called1 := false
	called2 := false

	s1 := CustomizeStrategy(func(ctx context.Context, target StrategyTarget) error {
		called1 = true
		return nil
	})
	s2 := CustomizeStrategy(func(ctx context.Context, target StrategyTarget) error {
		called2 = true
		return nil
	})

	composite := ForAll(s1, s2)
	err := composite.WaitUntilReady(context.Background(), &fakeTarget{})
	if err != nil {
		t.Fatalf("ForAll failed: %v", err)
	}
	if !called1 || !called2 {
		t.Errorf("expected both strategies to be called, got called1=%t, called2=%t", called1, called2)
	}
}

func TestForAnyStrategy(t *testing.T) {
	s1 := CustomizeStrategy(func(ctx context.Context, target StrategyTarget) error {
		return errors.New("failed")
	})
	s2 := CustomizeStrategy(func(ctx context.Context, target StrategyTarget) error {
		time.Sleep(10 * time.Millisecond)
		return nil
	})

	composite := ForAny(s1, s2)
	err := composite.WaitUntilReady(context.Background(), &fakeTarget{})
	if err != nil {
		t.Fatalf("ForAny failed: %v", err)
	}
}

func TestForLog(t *testing.T) {
	logContent := "line 1\nready log\nline 3\n"
	target := &fakeTarget{
		logsFunc: func(ctx context.Context) (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader(logContent)), nil
		},
	}

	strat := ForLog("ready log")
	err := strat.WaitUntilReady(context.Background(), target)
	if err != nil {
		t.Fatalf("ForLog failed: %v", err)
	}

	// Regex check
	stratRegex := ForLog("^ready.*$").AsRegexp()
	err = stratRegex.WaitUntilReady(context.Background(), target)
	if err != nil {
		t.Fatalf("ForLog regex failed: %v", err)
	}

	// Occurrence check
	stratOccur := ForLog("line").WithOccurrence(2)
	err = stratOccur.WaitUntilReady(context.Background(), target)
	if err != nil {
		t.Fatalf("ForLog occurrence failed: %v", err)
	}
}

func TestForListeningPort(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer func() { _ = listener.Close() }()

	_, portStr, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("failed to split host/port: %v", err)
	}
	portVal, _ := strconv.Atoi(portStr)

	target := &fakeTarget{
		hostFunc: func(ctx context.Context) (string, error) {
			return "127.0.0.1", nil
		},
		mappedPortFunc: func(ctx context.Context, port string) (int, error) {
			return portVal, nil
		},
	}

	strat := ForListeningPort("80")
	err = strat.WaitUntilReady(context.Background(), target)
	if err != nil {
		t.Fatalf("ForListeningPort failed: %v", err)
	}
}

func TestForHTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	u, _, _ := net.SplitHostPort(server.Listener.Addr().String())
	portVal, _ := strconv.Atoi(strings.Split(server.Listener.Addr().String(), ":")[1])

	target := &fakeTarget{
		hostFunc: func(ctx context.Context) (string, error) {
			return u, nil
		},
		mappedPortFunc: func(ctx context.Context, port string) (int, error) {
			return portVal, nil
		},
	}

	strat := ForHTTP("/").WithPort("80")
	err := strat.WaitUntilReady(context.Background(), target)
	if err != nil {
		t.Fatalf("ForHTTP failed: %v", err)
	}
}

func TestForExec(t *testing.T) {
	target := &fakeTarget{
		execFunc: func(ctx context.Context, cmd []string, opts ...any) (int, []byte, error) {
			if cmd[0] == "check" {
				return 0, []byte("ready-output"), nil
			}
			return 1, nil, nil
		},
	}

	strat := ForExec([]string{"check"}).WithResponseMatcher(func(r io.Reader) bool {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(r)
		return strings.Contains(buf.String(), "ready-output")
	})

	err := strat.WaitUntilReady(context.Background(), target)
	if err != nil {
		t.Fatalf("ForExec failed: %v", err)
	}
}

func TestForExit(t *testing.T) {
	count := 0
	target := &fakeTarget{
		statusFunc: func(ctx context.Context) (string, error) {
			count++
			if count < 3 {
				return "running", nil
			}
			return "exited", nil
		},
	}

	strat := ForExit()
	err := strat.WaitUntilReady(context.Background(), target)
	if err != nil {
		t.Fatalf("ForExit failed: %v", err)
	}
}

func TestForHealth(t *testing.T) {
	count := 0
	target := &fakeTarget{
		statusFunc: func(ctx context.Context) (string, error) {
			return "running", nil
		},
		exitCodeFunc: func(ctx context.Context) (int, error) {
			count++
			if count < 3 {
				return 1, nil
			}
			return 0, nil
		},
	}

	strat := ForHealth()
	err := strat.WaitUntilReady(context.Background(), target)
	if err != nil {
		t.Fatalf("ForHealth failed: %v", err)
	}
}

// Mock sql driver for testing ForSQL
type mockConn struct{ driver.Conn }

func (c *mockConn) Close() error { return nil }
func (c *mockConn) Begin() (driver.Tx, error) {
	return nil, errors.New("not implemented")
}
func (c *mockConn) Prepare(query string) (driver.Stmt, error) {
	return &mockStmt{query: query}, nil
}

type mockStmt struct {
	driver.Stmt
	query string
}

func (s *mockStmt) Close() error { return nil }
func (s *mockStmt) NumInput() int {
	return 0
}
func (s *mockStmt) Query(args []driver.Value) (driver.Rows, error) {
	return &mockRows{}, nil
}

type mockRows struct{ driver.Rows }

func (r *mockRows) Columns() []string {
	return []string{"col"}
}
func (r *mockRows) Close() error { return nil }
func (r *mockRows) Next(dest []driver.Value) error {
	return io.EOF
}

type mockDriver struct{}

func (d *mockDriver) Open(name string) (driver.Conn, error) {
	if name == "fail" {
		return nil, errors.New("connection failed")
	}
	return &mockConn{}, nil
}

var registerOnce sync.Once

func TestForSQL(t *testing.T) {
	registerOnce.Do(func() {
		sql.Register("mockdb", &mockDriver{})
	})

	dbCount := 0
	target := &fakeTarget{
		hostFunc: func(ctx context.Context) (string, error) {
			return "localhost", nil
		},
		mappedPortFunc: func(ctx context.Context, port string) (int, error) {
			return 3306, nil
		},
	}

	dburl := func(host string, port int) string {
		dbCount++
		if dbCount < 3 {
			return "fail"
		}
		return "success"
	}

	strat := ForSQL("3306", "mockdb", dburl)
	err := strat.WaitUntilReady(context.Background(), target)
	if err != nil {
		t.Fatalf("ForSQL failed: %v", err)
	}
}

func TestForFile(t *testing.T) {
	count := 0
	target := &fakeTarget{
		copyFileFunc: func(ctx context.Context, path string) (io.ReadCloser, error) {
			count++
			if count < 3 {
				return nil, errors.New("not ready yet")
			}
			return io.NopCloser(strings.NewReader("file content")), nil
		},
	}

	strat := ForFile("/ready")
	err := strat.WaitUntilReady(context.Background(), target)
	if err != nil {
		t.Fatalf("ForFile failed: %v", err)
	}
}

func TestPollStrategiesProbeImmediately(t *testing.T) {
	registerOnce.Do(func() {
		sql.Register("mockdb", &mockDriver{})
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	host, portString, err := net.SplitHostPort(server.Listener.Addr().String())
	require.NoError(t, err)
	port, err := strconv.Atoi(portString)
	require.NoError(t, err)

	tests := []struct {
		name     string
		strategy Strategy
		target   StrategyTarget
	}{
		{
			name:     "http",
			strategy: ForHTTP("/").WithPort("80").WithPollInterval(time.Hour),
			target: &fakeTarget{
				hostFunc:       func(context.Context) (string, error) { return host, nil },
				mappedPortFunc: func(context.Context, string) (int, error) { return port, nil },
			},
		},
		{
			name:     "port",
			strategy: ForListeningPort("80").WithPollInterval(time.Hour),
			target: &fakeTarget{
				hostFunc:       func(context.Context) (string, error) { return host, nil },
				mappedPortFunc: func(context.Context, string) (int, error) { return port, nil },
			},
		},
		{
			name:     "sql",
			strategy: ForSQL("3306", "mockdb", func(string, int) string { return "success" }).WithPollInterval(time.Hour),
			target: &fakeTarget{
				hostFunc:       func(context.Context) (string, error) { return "localhost", nil },
				mappedPortFunc: func(context.Context, string) (int, error) { return 3306, nil },
			},
		},
		{
			name:     "exec",
			strategy: ForExec([]string{"true"}).WithPollInterval(time.Hour),
			target:   &fakeTarget{},
		},
		{
			name:     "health",
			strategy: ForHealth().WithPollInterval(time.Hour),
			target:   &fakeTarget{},
		},
		{
			name:     "exit",
			strategy: ForExit().WithPollInterval(time.Hour),
			target: &fakeTarget{
				statusFunc: func(context.Context) (string, error) { return "exited", nil },
			},
		},
		{
			name:     "file",
			strategy: ForFile("/ready").WithPollInterval(time.Hour),
			target: &fakeTarget{
				copyFileFunc: func(context.Context, string) (io.ReadCloser, error) {
					return io.NopCloser(strings.NewReader("ready")), nil
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
			defer cancel()
			require.NoError(t, tt.strategy.WaitUntilReady(ctx, tt.target))
		})
	}
}

type CustomizeStrategy func(ctx context.Context, target StrategyTarget) error

func (c CustomizeStrategy) WaitUntilReady(ctx context.Context, target StrategyTarget) error {
	return c(ctx, target)
}

func TestWaitOptionsProper(t *testing.T) {
	dl := time.Second
	all := ForAll().WithDeadline(dl)
	assert.Equal(t, dl, all.Deadline)

	any := ForAny().WithDeadline(dl)
	assert.Equal(t, dl, any.Deadline)

	ex := ForExec([]string{"cmd"}).WithExitCodeMatcher(func(i int) bool { return i == 1 }).WithPollInterval(dl)
	assert.Equal(t, dl, ex.PollInterval)
	assert.True(t, ex.ExitCodeMatcher(1))
	assert.False(t, ex.ExitCodeMatcher(0))

	ext := ForExit().WithPollInterval(dl)
	assert.Equal(t, dl, ext.PollInterval)

	fl := ForFile("f").WithPollInterval(dl)
	assert.Equal(t, dl, fl.PollInterval)

	hl := ForHealth().WithPollInterval(dl)
	assert.Equal(t, dl, hl.PollInterval)

	ht := ForHTTP("").WithTLS().WithBasicAuth("u", "p").WithMethod("POST").
		WithStatusCodeMatcher(func(status int) bool { return status == 200 }).
		WithResponseMatcher(func(body io.Reader) bool { return true }).
		WithPollInterval(dl)
	assert.True(t, ht.UseTLS)
	assert.Equal(t, "POST", ht.Method)
	assert.Equal(t, dl, ht.PollInterval)

	lg := ForLog("").WithPollInterval(dl)
	assert.Equal(t, dl, lg.PollInterval)

	pt := ForListeningPort("").WithPollInterval(dl)
	assert.Equal(t, dl, pt.PollInterval)

	sq := ForSQL("80", "d", func(h string, p int) string { return "" }).WithQuery("q").WithPollInterval(dl)
	assert.Equal(t, dl, sq.PollInterval)
	assert.Equal(t, "q", sq.Query)
}

func TestForAllStrategy_Error(t *testing.T) {
	s1 := CustomizeStrategy(func(ctx context.Context, target StrategyTarget) error {
		return errors.New("sub-strategy error")
	})

	composite := ForAll(s1)
	err := composite.WaitUntilReady(context.Background(), &fakeTarget{})
	assert.ErrorContains(t, err, "sub-strategy error")
}

func TestForAnyStrategy_Error(t *testing.T) {
	s1 := CustomizeStrategy(func(ctx context.Context, target StrategyTarget) error {
		return errors.New("sub-strategy error")
	})

	composite := ForAny(s1)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err := composite.WaitUntilReady(ctx, &fakeTarget{})
	assert.Error(t, err)
}

func TestForHTTP_Error(t *testing.T) {
	strat := ForHTTP("/").WithPort("invalid").WithPollInterval(time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err := strat.WaitUntilReady(ctx, &fakeTarget{})
	assert.Error(t, err)

	strat2 := ForHTTP("/").WithPollInterval(time.Millisecond)
	err2 := strat2.WaitUntilReady(ctx, &fakeTarget{
		mappedPortFunc: func(ctx context.Context, port string) (int, error) {
			return 0, errors.New("mapped port error")
		},
	})
	assert.Error(t, err2)
}

func TestForSQL_Error(t *testing.T) {
	strat := ForSQL("80", "mockdb", func(h string, p int) string { return "fail" }).WithPollInterval(time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err := strat.WaitUntilReady(ctx, &fakeTarget{
		mappedPortFunc: func(ctx context.Context, port string) (int, error) {
			return 0, errors.New("mapped port error")
		},
	})
	assert.Error(t, err)
}

func TestForListeningPort_Error(t *testing.T) {
	strat := ForListeningPort("80").WithPollInterval(time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err := strat.WaitUntilReady(ctx, &fakeTarget{
		mappedPortFunc: func(ctx context.Context, port string) (int, error) {
			return 0, errors.New("mapped port error")
		},
	})
	assert.Error(t, err)
}

func TestForExec_Error(t *testing.T) {
	strat := ForExec([]string{"fail"}).WithPollInterval(time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err := strat.WaitUntilReady(ctx, &fakeTarget{
		execFunc: func(ctx context.Context, cmd []string, opts ...any) (int, []byte, error) {
			return 1, nil, errors.New("exec error")
		},
	})
	assert.Error(t, err)
}

func TestForExit_Error(t *testing.T) {
	strat := ForExit().WithPollInterval(time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err := strat.WaitUntilReady(ctx, &fakeTarget{
		statusFunc: func(ctx context.Context) (string, error) {
			return "", errors.New("status error")
		},
	})
	assert.NoError(t, err)
}

func TestForHealth_Error(t *testing.T) {
	strat := ForHealth().WithPollInterval(time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err := strat.WaitUntilReady(ctx, &fakeTarget{
		statusFunc: func(ctx context.Context) (string, error) {
			return "", errors.New("status error")
		},
	})
	assert.Error(t, err)
}

func TestForLog_Error(t *testing.T) {
	strat := ForLog("never").WithPollInterval(time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err := strat.WaitUntilReady(ctx, &fakeTarget{
		logsFunc: func(ctx context.Context) (io.ReadCloser, error) {
			return nil, errors.New("logs error")
		},
	})
	assert.Error(t, err)
}

func TestForFile_Error(t *testing.T) {
	strat := ForFile("never").WithPollInterval(time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err := strat.WaitUntilReady(ctx, &fakeTarget{
		copyFileFunc: func(ctx context.Context, path string) (io.ReadCloser, error) {
			return nil, errors.New("file error")
		},
	})
	assert.Error(t, err)
}

func TestForAllStrategy_Deadline(t *testing.T) {
	s1 := CustomizeStrategy(func(ctx context.Context, target StrategyTarget) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(1 * time.Second):
			return nil
		}
	})

	composite := ForAll(s1).WithDeadline(10 * time.Millisecond)
	err := composite.WaitUntilReady(context.Background(), &fakeTarget{})
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}
