# Build stage
FROM golang:1.25-alpine AS builder

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
    -o /build/aegisd ./cmd/aegisd/main.go

# Production runtime stage
FROM alpine:3.20

# Non-root user for Zero Trust security
RUN addgroup -S autorix && adduser -S autorix -G autorix

WORKDIR /app

# Copy certificates and compiled binary
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /build/aegisd /app/aegisd
# --chown so the non-root user can persist rule changes made through the
# admin API back to this file (see internal/rule.Store).
COPY --chown=autorix:autorix --from=builder /build/rules /app/rules

# Expose Proxy HTTP port and Admin REST API port
EXPOSE 4455 4456

# Run as non-root
USER autorix:autorix

ENTRYPOINT ["/app/aegisd"]
