package run_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/autorix/platform/run"
)

func TestRun_ShutsDownAllServersWhenContextCancelled(t *testing.T) {
	shutdownCalled := make(chan string, 2)

	newServer := func(name string) run.Named {
		block := make(chan struct{})
		return run.Named{
			Name: name,
			Serve: func() error {
				<-block
				return run.ErrServerClosed
			},
			Shutdown: func(ctx context.Context) error {
				shutdownCalled <- name
				close(block)
				return nil
			},
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	err := run.Run(ctx, 1*time.Second, nil, []run.Named{newServer("a"), newServer("b")})
	if err != nil {
		t.Fatalf("expected clean shutdown, got error: %v", err)
	}

	seen := map[string]bool{}
	for i := 0; i < 2; i++ {
		select {
		case name := <-shutdownCalled:
			seen[name] = true
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for both servers to shut down")
		}
	}
	if !seen["a"] || !seen["b"] {
		t.Fatalf("expected both servers to receive Shutdown, got %v", seen)
	}
}

func TestRun_ReturnsErrorWhenAServerCrashes(t *testing.T) {
	crashErr := errors.New("listener died")
	otherShutdown := make(chan struct{})

	crashing := run.Named{
		Name:     "crashing",
		Serve:    func() error { return crashErr },
		Shutdown: func(ctx context.Context) error { return nil },
	}
	healthy := run.Named{
		Name: "healthy",
		Serve: func() error {
			<-otherShutdown
			return run.ErrServerClosed
		},
		Shutdown: func(ctx context.Context) error {
			close(otherShutdown)
			return nil
		},
	}

	err := run.Run(context.Background(), 1*time.Second, nil, []run.Named{crashing, healthy})
	if !errors.Is(err, crashErr) {
		t.Fatalf("expected Run to surface the crashing server's error, got %v", err)
	}
}

func TestRun_ShutdownContextCarriesTheDrainBudget(t *testing.T) {
	var deadlineSeen time.Time
	block := make(chan struct{})

	server := run.Named{
		Name: "a",
		Serve: func() error {
			<-block
			return run.ErrServerClosed
		},
		Shutdown: func(ctx context.Context) error {
			deadlineSeen, _ = ctx.Deadline()
			close(block)
			return nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	start := time.Now()
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	if err := run.Run(ctx, 5*time.Second, nil, []run.Named{server}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if deadlineSeen.IsZero() {
		t.Fatalf("expected Shutdown to receive a context with a deadline")
	}
	budget := deadlineSeen.Sub(start)
	if budget < 4*time.Second || budget > 6*time.Second {
		t.Fatalf("expected the shutdown deadline to reflect the ~5s drain budget, got %v", budget)
	}
}

func TestRun_NoServersReturnsImmediatelyOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := run.Run(ctx, time.Second, nil, nil)
	if err != nil {
		t.Fatalf("expected no error for an empty server set, got %v", err)
	}
}
