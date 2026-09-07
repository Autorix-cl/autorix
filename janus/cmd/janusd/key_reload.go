package main

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/autorix/platform/run"
)

const defaultKeyReloadInterval = 30 * time.Second
const keyReloadTimeout = 10 * time.Second

// signingKeyReloader is deliberately limited to the operation needed by the
// process supervisor. It keeps the lifecycle runner independently testable.
type signingKeyReloader interface {
	Reload(context.Context) error
}

type reloadTicker interface {
	C() <-chan time.Time
	Stop()
}

type systemReloadTicker struct{ ticker *time.Ticker }

func (t *systemReloadTicker) C() <-chan time.Time { return t.ticker.C }
func (t *systemReloadTicker) Stop()               { t.ticker.Stop() }

type keyReloadRunner struct {
	reloader  signingKeyReloader
	interval  time.Duration
	logger    *slog.Logger
	newTicker func(time.Duration) reloadTicker

	ctx      context.Context
	cancel   context.CancelFunc
	stopOnce sync.Once
}

func newKeyReloadRunner(reloader signingKeyReloader, interval time.Duration, logger *slog.Logger) (*keyReloadRunner, error) {
	if reloader == nil {
		return nil, errors.New("signing key reloader is required")
	}
	if interval <= 0 {
		return nil, errors.New("persistent key reload interval must be positive")
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &keyReloadRunner{
		reloader: reloader,
		interval: interval,
		logger:   logger,
		newTicker: func(interval time.Duration) reloadTicker {
			return &systemReloadTicker{ticker: time.NewTicker(interval)}
		},
		ctx:    ctx,
		cancel: cancel,
	}, nil
}

// Serve periodically reloads signing keys created by peer Janus instances.
// A transient database failure must not stop token serving: the manager keeps
// its last known-good key set and retries on the next interval.
func (r *keyReloadRunner) Serve(parent context.Context) error {
	ticker := r.newTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-parent.Done():
			r.cancel()
			return run.ErrServerClosed
		case <-r.ctx.Done():
			return run.ErrServerClosed
		case <-ticker.C():
			reloadCtx, cancel := context.WithTimeout(r.ctx, keyReloadTimeout)
			err := r.reloader.Reload(reloadCtx)
			cancel()
			if err != nil && r.logger != nil {
				r.logger.Warn("failed to reload persistent signing keys; retaining last known-good key set", "error", err)
			}
		}
	}
}

func (r *keyReloadRunner) Shutdown(context.Context) error {
	r.stopOnce.Do(r.cancel)
	return nil
}
