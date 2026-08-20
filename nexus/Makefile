.PHONY: proto setup build run

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
BUILD_SHA ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_TIME ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -X github.com/autorix/platform/version.Version=$(VERSION) -X github.com/autorix/platform/version.BuildSHA=$(BUILD_SHA) -X github.com/autorix/platform/version.BuildTime=$(BUILD_TIME)

setup:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/bufbuild/buf/cmd/buf@latest

proto:
	PATH="$$(go env GOPATH)/bin:$$PATH" $$(go env GOPATH)/bin/buf generate

build:
	go build -ldflags "$(LDFLAGS)" -o bin/nexusd ./cmd/nexusd/main.go

run:
	go run ./cmd/nexusd/main.go
