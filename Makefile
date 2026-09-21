MODULE      := github.com/spacedreamer99/lodeen
BIN_DIR     := bin
GO          := go

VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT      ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE        ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS     := -s -w -X $(MODULE)/internal/shared/version.Version=$(VERSION) -X $(MODULE)/internal/shared/version.Commit=$(COMMIT) -X $(MODULE)/internal/shared/version.Date=$(DATE)

.PHONY: build
build: build-server build-client

.PHONY: build-server
build-server:
	$(GO) build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/lodeen-server ./cmd/server

.PHONY: build-client
build-client:
	$(GO) build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/lodeen-client ./cmd/client

.PHONY: test
test:
	$(GO) test -race -cover ./...

.PHONY: clean
clean:
	rm -rf $(BIN_DIR)
