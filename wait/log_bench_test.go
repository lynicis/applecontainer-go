package wait

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"
)

func BenchmarkLogStrategy(b *testing.B) {
	var data []byte
	for i := 1; i <= 10000; i++ {
		data = fmt.Appendf(data, "spam line %d\n", i)
	}
	data = append(data, "READY\n"...)

	target := &fakeTarget{
		logsFunc: func(context.Context) (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(data)), nil
		},
	}
	strategy := ForLog("READY")

	for b.Loop() {
		if err := strategy.WaitUntilReady(b.Context(), target); err != nil {
			b.Fatal(err)
		}
	}
}
