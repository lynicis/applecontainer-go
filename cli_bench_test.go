package applecontainer

import "testing"

var benchExecSink []byte

// BenchmarkExecRunner measures the process-spawn floor behind every CLI call.
func BenchmarkExecRunner(b *testing.B) {
	r := newExecRunner("/bin/echo")

	for b.Loop() {
		out, _, code, err := r.Run(b.Context(), []string{"bench"}, nil)
		if err != nil {
			b.Fatal(err)
		}
		if code != 0 {
			b.Fatalf("exit code: %d", code)
		}
		benchExecSink = out
	}
}
