# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Compile static binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -extldflags '-static'" \
    -o /build/egod ./cmd/egod/main.go

# Production runtime stage
FROM alpine:3.20

# Non-root user for Zero Trust security
RUN addgroup -S autorix && adduser -S autorix -G autorix

WORKDIR /app

# Copy certificates and compiled binary
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /build/egod /app/egod
COPY --from=builder /build/migrations /app/migrations
COPY --from=builder /build/schemas /app/schemas

# Expose HTTP port
EXPOSE 4433

# Run as non-root
USER autorix:autorix

ENTRYPOINT ["/app/egod"]
