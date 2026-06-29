package applecontainer

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/lynicis/applecontainer-go/log"
)

// logFanout manages streaming container logs to multiple consumers.
type logFanout struct {
	mu        sync.RWMutex
	consumers []LogConsumer
	cancel    context.CancelFunc
	rc        io.ReadCloser
	wg        sync.WaitGroup
	started   bool
}

// Start launches the background logs follower and streams logs.
func (lf *logFanout) Start(ctx context.Context, c *cliContainer) error {
	lf.mu.Lock()
	defer lf.mu.Unlock()

	if lf.started {
		return nil
	}

	// Register any consumers that are configured in c.req.LogConsumerCfg.
	if c.req.LogConsumerCfg != nil {
		lf.consumers = append(lf.consumers, c.req.LogConsumerCfg.Consumers...)
	}

	if len(lf.consumers) == 0 {
		return nil
	}

	subCtx, cancel := context.WithCancel(context.Background())
	lf.cancel = cancel

	rc, err := c.provider.ContainerLogs(subCtx, c.id, true, 0)
	if err != nil {
		cancel()
		return fmt.Errorf("applecontainer: failed to start log follower: %w", err)
	}
	lf.rc = rc

	lf.started = true
	lf.wg.Add(1)
	go func() {
		defer lf.wg.Done()
		scanner := bufio.NewScanner(rc)
		for scanner.Scan() {
			line := scanner.Bytes()
			content := make([]byte, len(line))
			copy(content, line)

			lf.mu.RLock()
			for _, consumer := range lf.consumers {
				consumer.Accept(Log{LogType: "stdout", Content: content})
			}
			lf.mu.RUnlock()
		}
		if err := scanner.Err(); err != nil {
			log.Printf("applecontainer: scanner error: %v", err)
		}
	}()

	return nil
}

// AddConsumer adds a consumer to the active log streams.
func (lf *logFanout) AddConsumer(consumer LogConsumer) {
	lf.mu.Lock()
	defer lf.mu.Unlock()
	lf.consumers = append(lf.consumers, consumer)
}

// RemoveConsumer removes a consumer.
func (lf *logFanout) RemoveConsumer(consumer LogConsumer) {
	lf.mu.Lock()
	defer lf.mu.Unlock()
	for i, c := range lf.consumers {
		if c == consumer {
			lf.consumers = append(lf.consumers[:i], lf.consumers[i+1:]...)
			break
		}
	}
}

// Stop terminates the logs follower process and scanner.
func (lf *logFanout) Stop() error {
	lf.mu.Lock()
	defer lf.mu.Unlock()

	if !lf.started {
		return nil
	}

	if lf.cancel != nil {
		lf.cancel()
	}
	if lf.rc != nil {
		_ = lf.rc.Close()
	}

	lf.started = false
	lf.wg.Wait()
	return nil
}

type closeWrapper struct {
	io.ReadCloser
	onClose func()
}

func (c *closeWrapper) Close() error {
	if c.onClose != nil {
		c.onClose()
	}
	return c.ReadCloser.Close()
}

type pipeConsumer struct {
	pw *io.PipeWriter
}

func (pc *pipeConsumer) Accept(l Log) {
	_, _ = pc.pw.Write(append(l.Content, '\n'))
}

// NewReader returns an io.ReadCloser that streams all logs printed from the fanout.
func (lf *logFanout) NewReader(ctx context.Context) (io.ReadCloser, error) {
	pr, pw := io.Pipe()
	consumer := &pipeConsumer{pw: pw}
	lf.AddConsumer(consumer)

	return &closeWrapper{
		ReadCloser: pr,
		onClose: func() {
			lf.RemoveConsumer(consumer)
			_ = pw.Close()
		},
	}, nil
}
