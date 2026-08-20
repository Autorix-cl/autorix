// Package run encapsulates signal handling, ordered start of multiple
// listeners and bounded graceful drain (P1-S2-T4). Nexus, Themis and Aegis
// each run two servers (HTTP admin + gRPC, or proxy + admin); this package
// removes the copy-pasted goroutine-and-signal-channel plumbing every
// engine's main.go otherwise duplicates.
package run

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// ErrServerClosed is the sentinel a Named.Serve function should return when
// it stopped because Shutdown was called, mirroring http.ErrServerClosed —
// Run does not treat it as a crash.
var ErrServerClosed = errors.New("run: server closed")

// Named pairs one listener (an *http.Server, a *grpc.Server, or anything
// else with a blocking serve loop and a graceful shutdown) with the name
// Run uses in logs and error messages.
type Named struct {
	Name string
	// Serve blocks until the server stops. It must return ErrServerClosed
	// when it stopped because Shutdown was called; any other error is
	// treated as a crash and triggers shutdown of every other server.
	Serve func() error
	// Shutdown drains in-flight requests and stops the server. ctx carries
	// the drain deadline Run was given.
	Shutdown func(ctx context.Context) error
}

// NotifyContext returns a context cancelled on SIGINT or SIGTERM, for
// production callers. Tests drive Run directly with a context they control.
func NotifyContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}

// Run starts every server concurrently and blocks until ctx is cancelled or
// one of them exits with an error other than ErrServerClosed. Either way,
// every server then receives Shutdown with a context bounded by drain. Run
// returns the triggering server's error, if any, after every Shutdown call
// has returned (best-effort — shutdown errors are logged, not returned,
// since the triggering error is what an operator needs to see first).
func Run(ctx context.Context, drain time.Duration, logger *slog.Logger, servers []Named) error {
	if len(servers) == 0 {
		<-ctx.Done()
		return nil
	}

	errs := make(chan error, len(servers))
	for _, s := range servers {
		s := s
		go func() {
			err := s.Serve()
			if err != nil && !errors.Is(err, ErrServerClosed) {
				errs <- &namedError{name: s.Name, err: err}
				return
			}
			errs <- nil
		}()
	}

	var triggerErr error
	consumed := 0
	select {
	case <-ctx.Done():
	case err := <-errs:
		triggerErr = err
		consumed = 1
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), drain)
	defer cancel()

	done := make(chan struct{})
	go func() {
		for _, s := range servers {
			if err := s.Shutdown(shutdownCtx); err != nil && logger != nil {
				logger.Error("shutdown error", "server", s.Name, "error", err)
			}
		}
		close(done)
	}()

	select {
	case <-done:
	case <-shutdownCtx.Done():
		if logger != nil {
			logger.Error("drain timeout exceeded; some servers may not have shut down cleanly")
		}
	}

	// Drain remaining Serve goroutines so they don't leak past Run.
	for i := consumed; i < len(servers); i++ {
		<-errs
	}

	return triggerErr
}

type namedError struct {
	name string
	err  error
}

func (e *namedError) Error() string { return e.name + ": " + e.err.Error() }
func (e *namedError) Unwrap() error { return e.err }
