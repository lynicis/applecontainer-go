.PHONY: lint test test-coverage test-benchmark bench-hermetic benchstat test-examples sec

GOLANGCI_LINT = go tool github.com/golangci/golangci-lint/v2/cmd/golangci-lint
GOSEC = go tool github.com/securego/gosec/v2/cmd/gosec

BENCH_COUNT ?= 5
BENCH_TIME  ?= 1s
BENCH_OLD   ?= old.txt
BENCH_NEW   ?= new.txt

lint:
	$(GOLANGCI_LINT) run ./...

fmt:
	$(GOLANGCI_LINT) fmt ./...

test:
	go test -race ./...

test-coverage:
	go test -v -race -coverprofile=coverage.txt ./...

test-benchmark:
	cd benchmarks && APPLECONTAINER_BENCHMARK=1 go test -bench=. -benchmem -benchtime=$(BENCH_TIME) -count=$(BENCH_COUNT) -timeout=3600s ./... > bench.txt
	cat benchmarks/bench.txt

bench-hermetic:
	go test -run '^$$' -bench=. -benchmem -count=$(BENCH_COUNT) ./...

benchstat:
	cd benchmarks && go tool benchstat $(BENCH_OLD) $(BENCH_NEW)

test-examples:
	cd examples && APPLECONTAINER_INTEGRATION=1 go test -tags integration -v ./...

sec:
	$(GOSEC) ./...
