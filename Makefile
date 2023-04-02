.PHONY: default
default: all

.PHONY: all
all: lint test

.PHONY: clean
clean:
	rm -f coverage.out

.PHONY: lint
lint:
	go vet ./...
	golangci-lint run

.PHONY: test
test:
	gotest -coverprofile coverage.out -race -timeout 10s
