# Build stage
FROM golang:1.25-alpine AS builder

ARG VERSION=dev
ARG BUILD_SHA=unknown
ARG BUILD_TIME=unknown

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Cache dependencies
COPY platform ./platform
COPY argus ./argus
COPY themis/go.mod themis/go.sum ./themis/
WORKDIR /build/themis
RUN go mod download
WORKDIR /build

# Copy source code
COPY themis ./themis
WORKDIR /build/themis

# Compile static binary with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -extldflags '-static' -X github.com/autorix/platform/version.Version=${VERSION} -X github.com/autorix/platform/version.BuildSHA=${BUILD_SHA} -X github.com/autorix/platform/version.BuildTime=${BUILD_TIME}" \
    -o /build/themisd ./cmd/themisd/main.go

# Production runtime stage
FROM alpine:3.20

# Create non-root user for Zero Trust security
RUN addgroup -S autorix && adduser -S autorix -G autorix

WORKDIR /app

# Copy certificates and compiled binary
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /build/themisd /app/themisd
COPY --from=builder /build/themis/migrations /app/migrations

# Expose gRPC and HTTP ports
EXPOSE 50052 4488

# Run as non-root
USER autorix:autorix

ENTRYPOINT ["/app/themisd"]
