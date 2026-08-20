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
COPY hermes/go.mod hermes/go.sum ./hermes/
WORKDIR /build/hermes
RUN go mod download
WORKDIR /build

# Copy source code
COPY hermes ./hermes
WORKDIR /build/hermes

# Compile static binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -extldflags '-static' -X github.com/autorix/platform/version.Version=${VERSION:-dev} -X github.com/autorix/platform/version.BuildSHA=${BUILD_SHA:-unknown} -X github.com/autorix/platform/version.BuildTime=${BUILD_TIME:-unknown}" \
    -o /build/hermesd ./cmd/hermesd/main.go

# Production runtime stage
FROM alpine:3.20

# Non-root user for Zero Trust security
RUN addgroup -S autorix && adduser -S autorix -G autorix

WORKDIR /app

# Copy certificates and compiled binary
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /build/hermesd /app/hermesd
COPY --from=builder /build/hermes/migrations /app/migrations

# Expose Hermes HTTP port
EXPOSE 4477

# Run as non-root
USER autorix:autorix

ENTRYPOINT ["/app/hermesd"]
