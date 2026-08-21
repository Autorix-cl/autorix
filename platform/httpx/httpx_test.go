package httpx_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/autorix/platform/httpx"
)

func TestRequestID_GeneratesWhenAbsentAndPropagatesInResponseHeader(t *testing.T) {
	var seen string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = httpx.RequestIDFromContext(r.Context())
	})

	h := httpx.RequestID(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if seen == "" {
		t.Fatalf("expected a generated request id in context")
	}
	if rec.Header().Get("X-Request-Id") != seen {
		t.Fatalf("expected response header X-Request-Id to echo the context value, got %q vs %q", rec.Header().Get("X-Request-Id"), seen)
	}
}

func TestRequestID_PropagatesIncomingHeaderInsteadOfGenerating(t *testing.T) {
	var seen string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = httpx.RequestIDFromContext(r.Context())
	})

	h := httpx.RequestID(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-Id", "client-supplied-id")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if seen != "client-supplied-id" {
		t.Fatalf("expected incoming request id to be preserved, got %q", seen)
	}
}

func TestRecover_CatchesPanicAndReturns500(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	})

	h := httpx.Recover(logger)(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req) // must not panic out of the handler

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 after a recovered panic, got %d", rec.Code)
	}
	if !strings.Contains(buf.String(), "boom") {
		t.Fatalf("expected the panic value to be logged, got: %s", buf.String())
	}
}

func TestAccessLog_LogsMethodPathStatusAndRequestID(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	h := httpx.RequestID(httpx.AccessLog(logger)(next))
	req := httptest.NewRequest(http.MethodPost, "/widgets", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("expected a JSON access log line, got %q: %v", buf.String(), err)
	}
	if entry["method"] != "POST" || entry["path"] != "/widgets" {
		t.Fatalf("expected method/path in access log, got %v", entry)
	}
	if status, ok := entry["status"].(float64); !ok || status != 201 {
		t.Fatalf("expected status=201 in access log, got %v", entry["status"])
	}
	if entry["request_id"] == nil || entry["request_id"] == "" {
		t.Fatalf("expected request_id propagated into the access log, got %v", entry["request_id"])
	}
}

func TestTimeout_CancelsContextAndReturns503WhenHandlerOverruns(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(200 * time.Millisecond):
			w.WriteHeader(http.StatusOK)
		case <-r.Context().Done():
			return
		}
	})

	h := httpx.Timeout(20 * time.Millisecond)(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when the handler overruns the timeout, got %d", rec.Code)
	}
	if rec.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %q", rec.Header().Get("Content-Type"))
	}
	expectedBody := `{"error":"request timed out"}`
	if strings.TrimSpace(rec.Body.String()) != expectedBody {
		t.Fatalf("expected JSON body %q, got %q", expectedBody, rec.Body.String())
	}
}

func TestTimeout_PassesThroughWhenHandlerIsFast(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	h := httpx.Timeout(200 * time.Millisecond)(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for a fast handler, got %d", rec.Code)
	}
}

func TestCORS_SetsHeadersForConfiguredOrigin(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	h := httpx.CORS(httpx.CORSConfig{AllowedOrigins: []string{"https://console.autorix.io"}})(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://console.autorix.io")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://console.autorix.io" {
		t.Fatalf("expected CORS origin header echoed, got %q", got)
	}
}

func TestCORS_HandlesPreflightWithoutCallingNext(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true })

	h := httpx.CORS(httpx.CORSConfig{AllowedOrigins: []string{"*"}})(next)
	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://console.autorix.io")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if called {
		t.Fatalf("expected preflight OPTIONS to short-circuit before reaching next")
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for preflight, got %d", rec.Code)
	}
}

func TestChain_AppliesMiddlewareInOrder(t *testing.T) {
	var order []string
	mw := func(name string) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, name)
				next.ServeHTTP(w, r)
			})
		}
	}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
	})

	h := httpx.Chain(next, mw("first"), mw("second"))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	want := []string{"first", "second", "handler"}
	if len(order) != len(want) {
		t.Fatalf("expected order %v, got %v", want, order)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("expected order %v, got %v", want, order)
		}
	}
}
