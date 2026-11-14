package applecontainer

import (
	"os"
	"testing"
)

var benchInspectSink *Inspect

func BenchmarkParseInspect(b *testing.B) {
	data, err := os.ReadFile("testdata/inspect.json")
	if err != nil {
		b.Fatal(err)
	}

	for b.Loop() {
		ins, err := parseInspect(data)
		if err != nil {
			b.Fatal(err)
		}
		benchInspectSink = ins
	}
}
