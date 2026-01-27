.PHONY: build install clean test fmt vet lint staticcheck check test-integration test-integration-build test-integration-rebuild test-integration-clean test-integration-up test-integration-down test-integration-logs test-integration-shell test-all

BINARY_NAME=bak
INSTALL_PATH=/usr/local/bin
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

build:
	go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/bak

install: build
	install -m 755 $(BINARY_NAME) $(INSTALL_PATH)/$(BINARY_NAME)

uninstall:
	rm -f $(INSTALL_PATH)/$(BINARY_NAME)

clean:
	rm -f $(BINARY_NAME)
	go clean

test:
	go test -v -race ./...

test-cover:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

fmt:
	go fmt ./...

fmt-check:
	@test -z "$$(gofmt -s -l .)" || (echo "Run 'make fmt' to format code" && gofmt -s -d . && exit 1)

vet:
	go vet ./...

staticcheck:
	@which staticcheck > /dev/null || go install honnef.co/go/tools/cmd/staticcheck@latest
	staticcheck ./...

lint: fmt vet staticcheck

check: fmt-check vet staticcheck test build
	@echo "All checks passed"

test-integration-build:
	docker build -f Dockerfile.test -t bak-integration-test .

test-integration-rebuild:
	docker build -f Dockerfile.test -t bak-integration-test --no-cache .

test-integration:
	./scripts/run-integration-tests.sh

test-integration-clean:
	docker rm -f bak-integration-tests 2>/dev/null || true
	docker rmi bak-integration-test 2>/dev/null || true

# Interactive debugging with docker compose
test-integration-up:
	docker compose -f docker-compose.test.yml up -d --build

test-integration-down:
	docker compose -f docker-compose.test.yml down -v

test-integration-logs:
	docker compose -f docker-compose.test.yml exec integration-tests journalctl -u integration-tests -f

test-integration-shell:
	docker compose -f docker-compose.test.yml exec integration-tests bash

test-all: test test-integration

.DEFAULT_GOAL := build
