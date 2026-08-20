package health

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Contract is the shared table-driven conformance suite from P1-S1-T10.
// Every engine imports it and runs it against its own real HTTP wiring, so
// the uniform contract is enforced by one shared assertion instead of seven
// hand-copied ones drifting apart.
//
// newMux must build the engine's real http.Handler — its actual
// transport/http server, mux and all — with its readiness check(s) rigged
// to fail iff checkErr is non-nil (e.g. a fake Pinger passed to
// platform/postgres.Check, or a Checker registered directly as in this
// package's own tests). Contract calls newMux(nil) and
// newMux(a non-nil error), never assuming which dependency backs the check.
func Contract(t *testing.T, newMux func(checkErr error) http.Handler) {
	t.Helper()

	t.Run("liveness stays green while a dependency is severed", func(t *testing.T) {
		mux := newMux(errors.New("connection refused"))
		rec := doGet(mux, "/health/alive")
		if rec.Code != http.StatusOK {
			t.Fatalf("liveness must never consult a dependency; expected 200, got %d", rec.Code)
		}
	})

	t.Run("readiness turns red while a dependency is severed", func(t *testing.T) {
		mux := newMux(errors.New("connection refused"))
		rec := doGet(mux, "/health/ready")
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("readiness must fail closed; expected 503, got %d", rec.Code)
		}
	})

	t.Run("all three endpoints respond when healthy", func(t *testing.T) {
		mux := newMux(nil)
		for _, path := range []string{"/health/alive", "/health/ready", "/info"} {
			rec := doGet(mux, path)
			if rec.Code != http.StatusOK {
				t.Errorf("%s: expected 200 when healthy, got %d", path, rec.Code)
			}
		}
	})
}

func doGet(mux http.Handler, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}
