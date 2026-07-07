package applecontainer

import (
	"os"
	"testing"
)

func BenchmarkParseInspect(b *testing.B) {
	data, err := os.ReadFile("testdata/inspect.json")
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := parseInspect(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseImageInspect(b *testing.B) {
	data := []byte(`[{"id": "sha256:dummy", "descriptor": {"size": 12345}, "reference": "nginx:latest"}]`)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := parseImageInspect(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}
