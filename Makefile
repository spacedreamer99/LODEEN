MODULE      := github.com/spacedreamer99/lodeen
BIN_DIR     := bin
GO          := go
IMAGE       ?= lodeen-server:dev

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

.PHONY: dev
dev:
	-./scripts/dev.sh

.PHONY: docker-build
docker-build:
	docker build --build-arg VERSION=$(VERSION) --build-arg COMMIT=$(COMMIT) --build-arg DATE=$(DATE) -t $(IMAGE) -f Dockerfile .

.PHONY: docker-run
docker-run:
	docker run --rm -p 7777:7777 -p 9091:9091 $(IMAGE)

.PHONY: docker-size
docker-size:
	docker images $(IMAGE)

.PHONY: clean
clean:
	rm -rf $(BIN_DIR) tmp

.PHONY: compose-up
compose-up:
	docker compose up -d --build

.PHONY: compose-down
compose-down:
	docker compose down

.PHONY: compose-logs
compose-logs:
	docker compose logs -f --tail=50

.PHONY: compose-ps
compose-ps:
	docker compose ps

.PHONY: compose-clean
compose-clean:
	docker compose down -v
