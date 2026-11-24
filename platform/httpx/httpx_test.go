package httpx_test

import (
	"bytes"
	"encoding/json"
	"io"
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
	if !strings.Contains(buf.String(), "stack") {
		t.Fatalf("expected stack trace to be logged, got: %s", buf.String())
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

func TestRequestID_RejectsInvalidAndOversizedHeaders(t *testing.T) {
	invalidCases := []string{
		"bad\nid",
		"bad\rid",
		"bad id with spaces",
		"bad<script>",
		strings.Repeat("a", 129),
	}

	for _, bad := range invalidCases {
		var seen string
		h := httpx.RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			seen = httpx.RequestIDFromContext(r.Context())
		}))
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Request-Id", bad)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if seen == "" || seen == bad {
			t.Fatalf("expected fresh generated request id for invalid input %q, got %q", bad, seen)
		}
		if rec.Header().Get("X-Request-Id") != seen {
			t.Fatalf("expected response header to match generated id")
		}
	}
}

func TestAccessLog_SupportsFlusherAndUnwrap(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	flushed := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f, ok := w.(http.Flusher)
		if !ok {
			t.Fatalf("expected ResponseWriter wrapped by AccessLog to implement http.Flusher")
		}
		f.Flush()
		if err := http.NewResponseController(w).Flush(); err != nil {
			t.Fatalf("expected ResponseController.Flush() to succeed: %v", err)
		}
		flushed = true
	})

	h := httpx.AccessLog(logger)(next)
	req := httptest.NewRequest(http.MethodGet, "/v1/stream", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if !flushed || !rec.Flushed {
		t.Fatalf("expected flush to propagate to underlying ResponseWriter")
	}
}

func TestCORS_WildcardSetsLiteralStarAndVaryOrigin(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	h := httpx.CORS(httpx.CORSConfig{AllowedOrigins: []string{"*"}})(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://attacker.example")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("expected literal '*' for wildcard CORS, got %q", got)
	}
	if got := rec.Header().Get("Vary"); !strings.Contains(got, "Origin") {
		t.Fatalf("expected Vary: Origin header, got %q", got)
	}
}

func TestCORS_DisallowedPreflightReturns403(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true })

	h := httpx.CORS(httpx.CORSConfig{AllowedOrigins: []string{"https://console.autorix.io"}})(next)
	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://evil.example")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if called {
		t.Fatalf("expected disallowed preflight not to reach next")
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for disallowed preflight, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected no Access-Control-Allow-Origin header, got %q", got)
	}
	if got := rec.Header().Get("Vary"); !strings.Contains(got, "Origin") {
		t.Fatalf("expected Vary: Origin header, got %q", got)
	}
}

func TestMaxBodyBytes_RejectsOversizedPayload(t *testing.T) {
	var readErr error
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, readErr = io.ReadAll(r.Body)
		if readErr != nil {
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	h := httpx.Chain(next, httpx.MaxBodyBytes(16))

	// Within limit
	reqOK := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("hello"))
	recOK := httptest.NewRecorder()
	h.ServeHTTP(recOK, reqOK)
	if recOK.Code != http.StatusOK || readErr != nil {
		t.Fatalf("expected small payload to succeed, got code=%d err=%v", recOK.Code, readErr)
	}

	// Exceeding limit
	reqBig := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(strings.Repeat("x", 64)))
	recBig := httptest.NewRecorder()
	h.ServeHTTP(recBig, reqBig)
	if recBig.Code != http.StatusRequestEntityTooLarge || readErr == nil {
		t.Fatalf("expected oversized payload to fail, got code=%d err=%v", recBig.Code, readErr)
	}
}
