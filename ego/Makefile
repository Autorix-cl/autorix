.PHONY: build run test

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
BUILD_SHA ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_TIME ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -X github.com/autorix/platform/version.Version=$(VERSION) -X github.com/autorix/platform/version.BuildSHA=$(BUILD_SHA) -X github.com/autorix/platform/version.BuildTime=$(BUILD_TIME)

build:
	go build -ldflags "$(LDFLAGS)" -o bin/egod ./cmd/egod/main.go

run:
	go run ./cmd/egod/main.go

test:
	go test -v ./...
