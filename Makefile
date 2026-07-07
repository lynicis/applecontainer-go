.PHONY: lint test test-coverage test-benchmark test-examples sec

lint:
	golangci-lint run ./...

test:
	go test -race ./...

test-coverage:
	go test -v -race -coverprofile=coverage.txt ./...

test-benchmark:
	cd benchmarks && APPLECONTAINER_BENCHMARK=1 go test -bench=. -benchtime=1x -tags benchmark -timeout=600s ./... && cd ../

test-examples:
	cd examples && go test -v -race ./... && cd ../

sec:
	gosec ./...
