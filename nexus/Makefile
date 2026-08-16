.PHONY: proto setup build run

setup:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/bufbuild/buf/cmd/buf@latest

proto:
	PATH="$$(go env GOPATH)/bin:$$PATH" $$(go env GOPATH)/bin/buf generate

build:
	go build -o bin/nexusd ./cmd/nexusd/main.go

run:
	go run ./cmd/nexusd/main.go
