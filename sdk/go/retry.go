package autorix

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"time"
)

// RetryConfig defines the exponential backoff with jitter retry strategy.
type RetryConfig struct {
	MaxRetries    int           // Maximum retry attempts (default: 3)
	InitialDelay  time.Duration // Initial retry delay (default: 50ms)
	MaxDelay      time.Duration // Maximum retry delay cap (default: 2s)
	BackoffFactor float64       // Exponential multiplier (default: 2.0)
}

// DefaultRetryConfig provides AWS/Google-standard retry parameters.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:    3,
		InitialDelay:  50 * time.Millisecond,
		MaxDelay:      2 * time.Second,
		BackoffFactor: 2.0,
	}
}

// isRetryableStatus returns true if the HTTP status code is transient and safe to retry.
func isRetryableStatus(statusCode int) bool {
	switch statusCode {
	case http.StatusTooManyRequests, // 429 Rate limited
		http.StatusInternalServerError, // 500
		http.StatusBadGateway,          // 502 Bad Gateway
		http.StatusServiceUnavailable,  // 503 Overloaded
		http.StatusGatewayTimeout:       // 504 Gateway Timeout
		return true
	default:
		return false
	}
}

// isRetryableError checks if a network/transport error is transient.
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout()
	}
	return false
}

// calculateJitterDelay computes Exponential Backoff with Full Jitter.
func calculateJitterDelay(attempt int, cfg RetryConfig) time.Duration {
	temp := float64(cfg.InitialDelay)
	for i := 0; i < attempt; i++ {
		temp *= cfg.BackoffFactor
		if temp > float64(cfg.MaxDelay) {
			temp = float64(cfg.MaxDelay)
			break
		}
	}

	maxMillis := int64(temp / float64(time.Millisecond))
	if maxMillis <= 1 {
		return cfg.InitialDelay
	}

	n, err := rand.Int(rand.Reader, big.NewInt(maxMillis))
	if err != nil {
		return time.Duration(maxMillis/2) * time.Millisecond
	}

	return time.Duration(n.Int64()) * time.Millisecond
}

// executeWithRetry executes an HTTP request operation with exponential backoff and jitter.
func executeWithRetry(
	ctx context.Context,
	cfg RetryConfig,
	operation func(ctx context.Context) (*http.Response, error),
) (*http.Response, error) {
	if cfg.MaxRetries <= 0 {
		return operation(ctx)
	}

	var lastErr error
	var lastResp *http.Response

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := calculateJitterDelay(attempt, cfg)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		resp, err := operation(ctx)
		if err != nil {
			lastErr = err
			if isRetryableError(err) && attempt < cfg.MaxRetries {
				continue
			}
			return nil, err
		}

		if !isRetryableStatus(resp.StatusCode) || attempt == cfg.MaxRetries {
			return resp, nil
		}

		// Drain and close response body before retrying
		_ = resp.Body.Close()
		lastResp = resp
	}

	if lastResp != nil {
		return lastResp, nil
	}
	return nil, fmt.Errorf("request failed after %d retries: %w", cfg.MaxRetries, lastErr)
}
