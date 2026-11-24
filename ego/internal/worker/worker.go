package worker

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	StatusPending     = "pending"
	StatusProcessing  = "processing"
	StatusDelivered   = "delivered"
	StatusDeadLetter  = "dead_letter"
	DefaultMaxRetries = 5
)

// Notification represents an email or SMS queued for delivery.
type Notification struct {
	ID            uuid.UUID              `json:"id"`
	Recipient     string                 `json:"recipient"`
	Template      string                 `json:"template"`
	Payload       map[string]interface{} `json:"payload"`
	Status        string                 `json:"status"`
	Attempts      int                    `json:"attempts"`
	MaxAttempts   int                    `json:"max_attempts"`
	LastError     *string                `json:"last_error,omitempty"`
	NextAttemptAt time.Time              `json:"next_attempt_at"`
	CreatedAt     time.Time              `json:"created_at"`
	DeliveredAt   *time.Time             `json:"delivered_at,omitempty"`
}

// Sender delivers a notification to an external provider (SMTP, SES, SMS).
type Sender interface {
	Send(ctx context.Context, n Notification) error
}

// OutboxRepository defines the persistence operations for notifications.
type OutboxRepository interface {
	EnqueueNotification(ctx context.Context, recipient, template string, payload map[string]interface{}) (uuid.UUID, error)
	FetchPendingNotifications(ctx context.Context, limit int) ([]Notification, error)
	MarkDelivered(ctx context.Context, id uuid.UUID) error
	MarkFailed(ctx context.Context, id uuid.UUID, lastError string, nextAttempt time.Time, deadLetter bool) error
}

// Worker polls the notification outbox and delivers notifications with exponential backoff.
type Worker struct {
	repo         OutboxRepository
	sender       Sender
	logger       *slog.Logger
	pollInterval time.Duration
	batchSize    int
	stopOnce     sync.Once
	stopChan     chan struct{}
	wg           sync.WaitGroup
}

// Config configures the background worker.
type Config struct {
	PollInterval time.Duration
	BatchSize    int
}

// New creates a new notification background worker.
func New(repo OutboxRepository, sender Sender, logger *slog.Logger, cfg Config) *Worker {
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 2 * time.Second
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 25
	}
	if logger == nil {
		logger = slog.Default()
	}

	return &Worker{
		repo:         repo,
		sender:       sender,
		logger:       logger,
		pollInterval: cfg.PollInterval,
		batchSize:    cfg.BatchSize,
		stopChan:     make(chan struct{}),
	}
}

// Start runs the worker in a background goroutine until ctx is cancelled.
func (w *Worker) Start(ctx context.Context) {
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		ticker := time.NewTicker(w.pollInterval)
		defer ticker.Stop()

		w.logger.Info("notification worker started", "poll_interval", w.pollInterval)

		for {
			select {
			case <-ctx.Done():
				w.logger.Info("notification worker shutting down")
				return
			case <-w.stopChan:
				return
			case <-ticker.C:
				if err := w.ProcessBatch(ctx); err != nil {
					w.logger.Error("error processing notification batch", "error", err)
				}
			}
		}
	}()
}

// Stop signals the worker to stop and waits for in-flight jobs to drain.
// Safe to call multiple times.
func (w *Worker) Stop() {
	w.stopOnce.Do(func() {
		close(w.stopChan)
	})
	w.wg.Wait()
}

// ProcessBatch fetches and processes one batch of pending notifications.
func (w *Worker) ProcessBatch(ctx context.Context) error {
	notifications, err := w.repo.FetchPendingNotifications(ctx, w.batchSize)
	if err != nil {
		return fmt.Errorf("fetch pending notifications: %w", err)
	}

	for _, n := range notifications {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		sendErr := w.sender.Send(ctx, n)
		if sendErr == nil {
			if err := w.repo.MarkDelivered(ctx, n.ID); err != nil {
				w.logger.Error("failed to mark notification delivered", "id", n.ID, "error", err)
			} else {
				w.logger.Info("notification delivered successfully", "id", n.ID, "recipient", n.Recipient, "template", n.Template)
			}
			continue
		}

		// Handle failure with exponential backoff
		newAttempts := n.Attempts + 1
		deadLetter := newAttempts >= n.MaxAttempts
		backoff := ComputeBackoff(newAttempts)
		nextAttempt := time.Now().Add(backoff)

		if err := w.repo.MarkFailed(ctx, n.ID, sendErr.Error(), nextAttempt, deadLetter); err != nil {
			w.logger.Error("failed to mark notification failed", "id", n.ID, "error", err)
		} else {
			if deadLetter {
				w.logger.Warn("notification moved to dead letter queue", "id", n.ID, "attempts", newAttempts, "error", sendErr)
			} else {
				w.logger.Info("notification scheduled for retry", "id", n.ID, "next_attempt", nextAttempt, "attempts", newAttempts)
			}
		}
	}

	return nil
}

// ComputeBackoff calculates exponential backoff with jitter: base * 2^(attempt-1) + jitter.
func ComputeBackoff(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}

	base := 1 * time.Second
	maxBackoff := 5 * time.Minute

	shift := attempt - 1
	if shift > 16 {
		shift = 16
	}
	backoff := base * (1 << uint(shift))
	if backoff > maxBackoff || backoff <= 0 {
		backoff = maxBackoff
	}

	// Add up to 25% jitter to prevent thundering herd
	jitterBound := int64(backoff / 4)
	if jitterBound <= 0 {
		return backoff
	}
	jitter := time.Duration(rand.Int63n(jitterBound))
	return backoff + jitter
}

// LoggingSender is a reference sender that logs emails/SMS with sensitive fields redacted.
type LoggingSender struct {
	logger *slog.Logger
}

func NewLoggingSender(logger *slog.Logger) *LoggingSender {
	if logger == nil {
		logger = slog.Default()
	}
	return &LoggingSender{logger: logger}
}

func redactNotificationPayload(payload map[string]interface{}) map[string]interface{} {
	if payload == nil {
		return nil
	}
	sanitized := make(map[string]interface{}, len(payload))
	for k, v := range payload {
		lower := strings.ToLower(k)
		if strings.Contains(lower, "token") ||
			strings.Contains(lower, "secret") ||
			strings.Contains(lower, "password") ||
			strings.Contains(lower, "recovery_link") ||
			strings.Contains(lower, "link") ||
			strings.Contains(lower, "url") ||
			strings.Contains(lower, "code") {
			sanitized[k] = "[REDACTED]"
			continue
		}
		sanitized[k] = v
	}
	return sanitized
}

func (s *LoggingSender) Send(ctx context.Context, n Notification) error {
	s.logger.Info("DISPATCH TRANSACTIONAL NOTIFICATION",
		"recipient", n.Recipient,
		"template", n.Template,
		"payload", redactNotificationPayload(n.Payload),
	)
	return nil
}
