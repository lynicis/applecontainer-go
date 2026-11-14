package wait

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func BenchmarkHTTPStrategy(b *testing.B) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	addr := srv.Listener.Addr().(*net.TCPAddr)
	target := &fakeTarget{
		hostFunc:       func(context.Context) (string, error) { return addr.IP.String(), nil },
		mappedPortFunc: func(context.Context, string) (int, error) { return addr.Port, nil },
	}
	strategy := ForHTTP("/").WithPollInterval(time.Millisecond)

	for b.Loop() {
		if err := strategy.WaitUntilReady(b.Context(), target); err != nil {
			b.Fatal(err)
		}
	}
}
