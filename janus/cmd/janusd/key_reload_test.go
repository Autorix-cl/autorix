package main

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/autorix/platform/config"
	"github.com/autorix/platform/run"
)

type testReloadTicker struct{ ticks chan time.Time }

func newTestReloadTicker() *testReloadTicker {
	return &testReloadTicker{ticks: make(chan time.Time, 4)}
}
func (t *testReloadTicker) C() <-chan time.Time { return t.ticks }
func (t *testReloadTicker) Stop()               {}

type testSigningKeyReloader struct {
	mu       sync.Mutex
	calls    int
	failures int
	done     chan struct{}
}

func (r *testSigningKeyReloader) Reload(context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls++
	select {
	case r.done <- struct{}{}:
	default:
	}
	if r.failures > 0 {
		r.failures--
		return errors.New("database unavailable")
	}
	return nil
}

func TestPersistentKeyReloadIntervalDefaultAndValidation(t *testing.T) {
	previous, wasSet := os.LookupEnv("KEY_RELOAD_INTERVAL")
	if err := os.Unsetenv("KEY_RELOAD_INTERVAL"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if wasSet {
			_ = os.Setenv("KEY_RELOAD_INTERVAL", previous)
			return
		}
		_ = os.Unsetenv("KEY_RELOAD_INTERVAL")
	})
	var cfg appConfig
	if err := config.Load(&cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.KeyReloadInterval != defaultKeyReloadInterval {
		t.Fatalf("default reload interval = %s, want %s", cfg.KeyReloadInterval, defaultKeyReloadInterval)
	}

	for _, interval := range []time.Duration{0, -time.Second} {
		cfg := appConfig{AdminHost: "127.0.0.1", AdminPort: "4445", KeyReloadInterval: interval}
		if err := cfg.validate(); err == nil {
			t.Fatalf("validate accepted interval %s", interval)
		}
	}
}

func TestKeyReloadRunnerRetriesAfterFailureAndStopsCleanly(t *testing.T) {
	reloader := &testSigningKeyReloader{failures: 1, done: make(chan struct{}, 2)}
	runner, err := newKeyReloadRunner(reloader, time.Second, nil)
	if err != nil {
		t.Fatal(err)
	}
	ticker := newTestReloadTicker()
	runner.newTicker = func(time.Duration) reloadTicker { return ticker }

	finished := make(chan error, 1)
	go func() { finished <- runner.Serve(context.Background()) }()

	// The first failure must be contained; the next scheduled reload still runs.
	ticker.ticks <- time.Now()
	awaitReload(t, reloader.done)
	ticker.ticks <- time.Now()
	awaitReload(t, reloader.done)

	if err := runner.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-finished:
		if !errors.Is(err, run.ErrServerClosed) {
			t.Fatalf("runner exit = %v, want ErrServerClosed", err)
		}
	case <-time.After(time.Second):
		t.Fatal("key reload runner did not stop")
	}
}

func TestNewKeyReloadRunnerRejectsInvalidDependencies(t *testing.T) {
	if _, err := newKeyReloadRunner(nil, time.Second, nil); err == nil {
		t.Fatal("nil reloader was accepted")
	}
	if _, err := newKeyReloadRunner(&testSigningKeyReloader{}, 0, nil); err == nil {
		t.Fatal("zero interval was accepted")
	}
}

func awaitReload(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for signing-key reload")
	}
}
