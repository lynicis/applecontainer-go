package applecontainer

import (
	"bufio"
	"context"
	"io"
	"os/exec"
	"sync"
	"testing"
	"time"
)

type mockLogConsumer struct {
	mu   sync.Mutex
	logs []string
}

func (m *mockLogConsumer) Accept(l Log) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, string(l.Content))
}

func (m *mockLogConsumer) getLogs() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.logs
}

func TestLogFanout(t *testing.T) {
	pr, pw := io.Pipe()

	runner := &fakeRunner{
		startFn: func(ctx context.Context, args []string, stdin io.Reader) (*exec.Cmd, io.Reader, io.Reader, error) {
			cmd := exec.Command("sleep", "10")
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			return cmd, pr, nil, nil
		},
	}

	p := &cliProvider{
		runner: runner,
		cfg:    Config{},
	}

	c := &cliContainer{
		provider: p,
		id:       "test-container-id",
	}

	lf := &logFanout{}
	consumer := &mockLogConsumer{}
	lf.AddConsumer(consumer)

	// Obtain a stream reader via NewReader
	r, err := lf.NewReader(context.Background())
	if err != nil {
		t.Fatalf("failed to create reader: %v", err)
	}

	err = lf.Start(context.Background(), c)
	if err != nil {
		t.Fatalf("failed to start logFanout: %v", err)
	}

	// Write logs to provider log pipe
	go func() {
		_, _ = pw.Write([]byte("first line\nsecond line\n"))
		_ = pw.Close()
	}()

	// Verify reader gets the lines cleanly
	br := bufio.NewReader(r)
	l1, err := br.ReadString('\n')
	if err != nil {
		t.Fatalf("failed to read line 1: %v", err)
	}
	l2, err := br.ReadString('\n')
	if err != nil {
		t.Fatalf("failed to read line 2: %v", err)
	}

	if l1 != "first line\n" || l2 != "second line\n" {
		t.Errorf("unexpected read content: %q then %q", l1, l2)
	}

	// Wait briefly to allow async fanout to consumer
	time.Sleep(20 * time.Millisecond)

	logs := consumer.getLogs()
	if len(logs) < 2 {
		t.Fatalf("expected at least 2 logs, got %v", logs)
	}
	if logs[0] != "first line" || logs[1] != "second line" {
		t.Errorf("unexpected logs in consumer: %v", logs)
	}

	_ = r.Close()
	_ = lf.Stop()
}
