.PHONY: proto setup build run test bench

setup:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/bufbuild/buf/cmd/buf@latest

proto:
	PATH="$$(go env GOPATH)/bin:$$PATH" $$(go env GOPATH)/bin/buf generate

build:
	go build -o bin/themisd ./cmd/themisd/main.go

run:
	go run ./cmd/themisd/main.go

test:
	go test ./... -v -count=1

bench:
	go test ./internal/engine/ -bench=. -benchmem -count=3
