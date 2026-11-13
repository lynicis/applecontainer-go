.PHONY: lint test test-coverage test-benchmark test-examples sec

GOLANGCI_LINT = go tool github.com/golangci/golangci-lint/v2/cmd/golangci-lint
GOSEC = go tool github.com/securego/gosec/v2/cmd/gosec

lint:
	$(GOLANGCI_LINT) run ./...

fmt:
	$(GOLANGCI_LINT) fmt ./...

test:
	go test -race ./...

test-coverage:
	go test -v -race -coverprofile=coverage.txt ./...

test-benchmark:
	cd benchmarks && APPLECONTAINER_BENCHMARK=1 go test -bench=. -benchtime=1x -tags benchmark -timeout=600s ./... && cd ../

test-examples:
	cd examples && APPLECONTAINER_INTEGRATION=1 go test -tags integration -v ./... && cd ../

sec:
	$(GOSEC) ./...
