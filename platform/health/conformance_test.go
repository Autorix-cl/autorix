package health_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/autorix/platform/health"
)

// newConformingMux builds a mux exactly the way an engine's main.go does:
// register the platform handler over a checker with one named check whose
// outcome the test controls. It satisfies health.Contract's newMux shape.
func newConformingMux(checkErr error) http.Handler {
	c := health.NewChecker()
	c.Register("postgres", func(ctx context.Context) error { return checkErr })
	h := health.NewHandler(c, func() health.Info {
		return health.Info{Engine: "conformance-fixture", Version: "0.0.0-test"}
	})
	mux := http.NewServeMux()
	h.Register(mux)
	return mux
}

// TestContract_PassesForAConformingEngine proves health.Contract itself is
// usable and passes for a correctly wired engine — the shape every one of
// the 7 engines' own test suites now import and run against their real
// server construction (P1-S1-T10).
func TestContract_PassesForAConformingEngine(t *testing.T) {
	health.Contract(t, newConformingMux)
}
