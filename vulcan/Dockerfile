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
COPY vulcan/go.mod vulcan/go.sum ./vulcan/
WORKDIR /build/vulcan
RUN go mod download
WORKDIR /build

# Copy source code
COPY vulcan ./vulcan
WORKDIR /build/vulcan

# Compile static binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -extldflags '-static' -X github.com/autorix/platform/version.Version=${VERSION:-dev} -X github.com/autorix/platform/version.BuildSHA=${BUILD_SHA:-unknown} -X github.com/autorix/platform/version.BuildTime=${BUILD_TIME:-unknown}" \
    -o /build/vulcand ./cmd/vulcand/main.go

# Production runtime stage
FROM alpine:3.20

# Non-root user for Zero Trust security
RUN addgroup -S autorix && adduser -S autorix -G autorix

WORKDIR /app

# Copy certificates and compiled binary
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /build/vulcand /app/vulcand
COPY --from=builder /build/vulcan/migrations /app/migrations

# Expose Vulcan HTTP port
EXPOSE 4466

# Run as non-root
USER autorix:autorix

ENTRYPOINT ["/app/vulcand"]
