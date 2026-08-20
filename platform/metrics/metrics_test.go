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
