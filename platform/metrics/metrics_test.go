package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMetrics_HandlerAndMiddleware(t *testing.T) {
	middleware := HTTPMiddleware("test-engine")

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest("GET", "/test-path", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	metricsReq := httptest.NewRequest("GET", "/metrics", nil)
	metricsRec := httptest.NewRecorder()
	Handler().ServeHTTP(metricsRec, metricsReq)

	if metricsRec.Code != http.StatusOK {
		t.Fatalf("expected metrics status 200, got %d", metricsRec.Code)
	}

	body := metricsRec.Body.String()
	if !strings.Contains(body, "autorix_http_requests_total") {
		t.Errorf("expected metrics body to contain autorix_http_requests_total, got: %s", body)
	}
}

func TestMetrics_RegisterPoolStatsNil(t *testing.T) {
	// Should not panic on nil pool
	RegisterPoolStats("test-engine", nil)
	UpdatePostgresPoolStats("test-engine")
}

func TestMetrics_HTTPMiddleware_UsesRoutePatternAndBoundedFallback(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /items/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := HTTPMiddleware("cardinality-engine")(mux)

	// Parameterized route must use the route pattern, not raw IDs.
	for _, rawPath := range []string{"/items/alpha-1", "/items/beta-2"} {
		req := httptest.NewRequest(http.MethodGet, rawPath, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}

	// Unmatched arbitrary paths must collapse to "unmatched".
	req404 := httptest.NewRequest(http.MethodGet, "/random-attack-9f8e7d6c", nil)
	rec404 := httptest.NewRecorder()
	handler.ServeHTTP(rec404, req404)

	metricsRec := httptest.NewRecorder()
	Handler().ServeHTTP(metricsRec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := metricsRec.Body.String()

	if strings.Contains(body, "/items/alpha-1") || strings.Contains(body, "/random-attack-9f8e7d6c") {
		t.Fatalf("expected raw request paths not to appear in metrics labels, got: %s", body)
	}
	if !strings.Contains(body, `path="GET /items/{id}"`) {
		t.Fatalf("expected matched route pattern label in metrics, got: %s", body)
	}
	if !strings.Contains(body, `path="unmatched"`) {
		t.Fatalf("expected unmatched fallback label in metrics, got: %s", body)
	}
}

func TestMetrics_LivePostgresPoolCollectorUpdatesOnScrape(t *testing.T) {
	current := poolStatSnapshot{total: 2, idle: 2, acquired: 0, max: 10}
	registerPoolStatProvider("live-pool-engine", func() poolStatSnapshot {
		return current
	})
	t.Cleanup(func() {
		registeredPoolsMu.Lock()
		delete(registeredPools, "live-pool-engine")
		registeredPoolsMu.Unlock()
	})

	scrape := func() string {
		rec := httptest.NewRecorder()
		Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
		return rec.Body.String()
	}

	first := scrape()
	if !strings.Contains(first, `autorix_postgres_pool_acquired_conns{engine="live-pool-engine"} 0`) {
		t.Fatalf("expected acquired=0 on first scrape, got: %s", first)
	}

	current = poolStatSnapshot{total: 5, idle: 1, acquired: 4, max: 10}
	second := scrape()
	if !strings.Contains(second, `autorix_postgres_pool_acquired_conns{engine="live-pool-engine"} 4`) {
		t.Fatalf("expected acquired=4 on second scrape, got: %s", second)
	}
	if !strings.Contains(second, `autorix_postgres_pool_total_conns{engine="live-pool-engine"} 5`) {
		t.Fatalf("expected total=5 on second scrape, got: %s", second)
	}
}
