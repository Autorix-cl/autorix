package worker

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

type mockOutboxRepo struct {
	mu            sync.Mutex
	notifications map[uuid.UUID]*Notification
}

func newMockOutboxRepo() *mockOutboxRepo {
	return &mockOutboxRepo{notifications: make(map[uuid.UUID]*Notification)}
}

func (m *mockOutboxRepo) EnqueueNotification(ctx context.Context, recipient, template string, payload map[string]interface{}) (uuid.UUID, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := uuid.New()
	m.notifications[id] = &Notification{
		ID:            id,
		Recipient:     recipient,
		Template:      template,
		Payload:       payload,
		Status:        StatusPending,
		Attempts:      0,
		MaxAttempts:   DefaultMaxRetries,
		NextAttemptAt: time.Now(),
		CreatedAt:     time.Now(),
	}
	return id, nil
}

func (m *mockOutboxRepo) FetchPendingNotifications(ctx context.Context, limit int) ([]Notification, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var res []Notification
	for _, n := range m.notifications {
		if n.Status == StatusPending && !n.NextAttemptAt.After(time.Now()) {
			res = append(res, *n)
			if len(res) >= limit {
				break
			}
		}
	}
	return res, nil
}

func (m *mockOutboxRepo) MarkDelivered(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if n, ok := m.notifications[id]; ok {
		n.Status = StatusDelivered
		now := time.Now()
		n.DeliveredAt = &now
		return nil
	}
	return errors.New("not found")
}

func (m *mockOutboxRepo) MarkFailed(ctx context.Context, id uuid.UUID, lastError string, nextAttempt time.Time, deadLetter bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if n, ok := m.notifications[id]; ok {
		n.Attempts++
		n.LastError = &lastError
		n.NextAttemptAt = nextAttempt
		if deadLetter {
			n.Status = StatusDeadLetter
		} else {
			n.Status = StatusPending
		}
		return nil
	}
	return errors.New("not found")
}

type mockSender struct {
	mu        sync.Mutex
	failUntil int
	attempts  int
}

func (s *mockSender) Send(ctx context.Context, n Notification) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.attempts++
	if s.attempts <= s.failUntil {
		return errors.New("transient SMTP failure")
	}
	return nil
}

func TestWorker_SuccessfulDelivery(t *testing.T) {
	repo := newMockOutboxRepo()
	sender := &mockSender{failUntil: 0}
	w := New(repo, sender, nil, Config{BatchSize: 10})

	id, err := repo.EnqueueNotification(context.Background(), "user@example.com", "recovery", map[string]interface{}{"token": "abc"})
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	if err := w.ProcessBatch(context.Background()); err != nil {
		t.Fatalf("process batch: %v", err)
	}

	repo.mu.Lock()
	n := repo.notifications[id]
	repo.mu.Unlock()

	if n.Status != StatusDelivered {
		t.Errorf("status = %s, want %s", n.Status, StatusDelivered)
	}
	if n.DeliveredAt == nil {
		t.Error("expected delivered_at timestamp to be set")
	}
}

func TestWorker_RetryAndDeadLetterQueue(t *testing.T) {
	repo := newMockOutboxRepo()
	sender := &mockSender{failUntil: 999} // Always fails
	w := New(repo, sender, nil, Config{BatchSize: 10})

	id, _ := repo.EnqueueNotification(context.Background(), "user@example.com", "recovery", map[string]interface{}{})

	// Process 1st attempt
	if err := w.ProcessBatch(context.Background()); err != nil {
		t.Fatalf("process batch: %v", err)
	}

	repo.mu.Lock()
	n := repo.notifications[id]
	repo.mu.Unlock()

	if n.Status != StatusPending {
		t.Errorf("status after 1 failure = %s, want %s", n.Status, StatusPending)
	}
	if n.Attempts != 1 {
		t.Errorf("attempts = %d, want 1", n.Attempts)
	}

	// Fast forward attempts to MaxAttempts
	for i := 2; i <= DefaultMaxRetries; i++ {
		// artificially set nextAttempt to now so it gets picked up
		repo.mu.Lock()
		n.NextAttemptAt = time.Now().Add(-1 * time.Second)
		repo.mu.Unlock()

		_ = w.ProcessBatch(context.Background())
	}

	repo.mu.Lock()
	n = repo.notifications[id]
	repo.mu.Unlock()

	if n.Status != StatusDeadLetter {
		t.Errorf("status after max retries = %s, want %s", n.Status, StatusDeadLetter)
	}
	if n.Attempts != DefaultMaxRetries {
		t.Errorf("attempts = %d, want %d", n.Attempts, DefaultMaxRetries)
	}
}

func TestComputeBackoff(t *testing.T) {
	b1 := ComputeBackoff(1)
	b2 := ComputeBackoff(2)
	b3 := ComputeBackoff(3)

	if b1 <= 0 || b2 <= b1 || b3 <= b2 {
		t.Errorf("backoffs not strictly monotonic: b1=%v, b2=%v, b3=%v", b1, b2, b3)
	}
}
