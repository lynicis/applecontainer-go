.PHONY: lint test test-coverage test-benchmark test-examples sec

golangci-lint = "go tool github.com/golangci/golangci-lint/v2/cmd/golangci-lint"
gosec = "go tool github.com/securego/gosec/v2/cmd/gosec"

lint:
	golangci-lint run ./...

fmt:
	golangci-lint fmt ./...

test:
	go test -race ./...

test-coverage:
	go test -v -race -coverprofile=coverage.txt ./...

test-benchmark:
	cd benchmarks && APPLECONTAINER_BENCHMARK=1 go test -bench=. -benchtime=1x -tags benchmark -timeout=600s ./... && cd ../

test-examples:
	cd examples && APPLECONTAINER_INTEGRATION=1 go test -tags integration -v ./... && cd ../

sec:
	gosec ./...
