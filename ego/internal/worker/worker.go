package worker

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	StatusPending    = "pending"
	StatusDelivered  = "delivered"
	StatusDeadLetter = "dead_letter"
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
func (w *Worker) Stop() {
	close(w.stopChan)
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

// ComputeBackoff calculates exponential backoff with jitter: base * 2^attempt + jitter.
func ComputeBackoff(attempt int) time.Duration {
	base := 1 * time.Second
	maxBackoff := 5 * time.Minute

	backoff := base * (1 << uint(attempt-1))
	if backoff > maxBackoff || backoff <= 0 {
		backoff = maxBackoff
	}

	// Add 10-25% jitter to prevent thundering herd
	jitter := time.Duration(rand.Int63n(int64(backoff / 4)))
	return backoff + jitter
}

// LoggingSender is a reference sender that logs emails/SMS.
type LoggingSender struct {
	logger *slog.Logger
}

func NewLoggingSender(logger *slog.Logger) *LoggingSender {
	if logger == nil {
		logger = slog.Default()
	}
	return &LoggingSender{logger: logger}
}

func (s *LoggingSender) Send(ctx context.Context, n Notification) error {
	s.logger.Info("DISPATCH TRANSACTIONAL NOTIFICATION",
		"recipient", n.Recipient,
		"template", n.Template,
		"payload", n.Payload,
	)
	return nil
}
