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
COPY nexus/go.mod nexus/go.sum ./nexus/
WORKDIR /build/nexus
RUN go mod download
WORKDIR /build

# Copy source code
COPY nexus ./nexus
WORKDIR /build/nexus

# Compile static binary with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -extldflags '-static' -X github.com/autorix/platform/version.Version=${VERSION} -X github.com/autorix/platform/version.BuildSHA=${BUILD_SHA} -X github.com/autorix/platform/version.BuildTime=${BUILD_TIME}" \
    -o /build/nexusd ./cmd/nexusd/main.go

# Production runtime stage
FROM alpine:3.20

# Create non-root user for Zero Trust security
RUN addgroup -S autorix && adduser -S autorix -G autorix

WORKDIR /app

# Copy certificates and compiled binary
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /build/nexusd /app/nexusd
COPY --from=builder /build/nexus/migrations /app/migrations

# Expose gRPC port and REST admin API port
EXPOSE 50051 8080

# Run as non-root
USER autorix:autorix

ENTRYPOINT ["/app/nexusd"]
