.PHONY: build install test lint fmt clean

BINARY = mtgo-cli
LDFLAGS = -X main.version=$(shell git describe --tags --always 2>/dev/null || echo "dev") \
          -X main.commit=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown") \
          -X main.buildTime=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/mtgo-cli/

install:
	go install -ldflags "$(LDFLAGS)" ./cmd/mtgo-cli/

test:
	go test ./...

lint:
	golangci-lint run ./...

fmt:
	golangci-lint fmt ./...

clean:
	rm -f $(BINARY)
