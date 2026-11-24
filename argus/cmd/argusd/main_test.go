package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTimeoutExceptStream_BypassesTimeoutForStreamAndTimesOutOtherRoutes(t *testing.T) {
	mw := timeoutExceptStream(25 * time.Millisecond)

	streamHandler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "flusher missing", http.StatusInternalServerError)
			return
		}
		time.Sleep(60 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("event: instances\ndata: []\n\n"))
		flusher.Flush()
	}))

	streamReq := httptest.NewRequest(http.MethodGet, "/v1/stream", nil)
	streamRec := httptest.NewRecorder()
	streamHandler.ServeHTTP(streamRec, streamReq)
	if streamRec.Code != http.StatusOK {
		t.Fatalf("expected /v1/stream to bypass timeout and preserve Flusher, got %d: %s", streamRec.Code, streamRec.Body.String())
	}

	normalHandler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(150 * time.Millisecond):
			w.WriteHeader(http.StatusOK)
		case <-r.Context().Done():
			return
		}
	}))

	normalReq := httptest.NewRequest(http.MethodGet, "/v1/instances", nil)
	normalRec := httptest.NewRecorder()
	normalHandler.ServeHTTP(normalRec, normalReq)
	if normalRec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected non-stream route to time out with 503, got %d", normalRec.Code)
	}
}
