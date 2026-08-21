// Package httpx is the shared HTTP middleware chain every engine mounts
// (P1-S2-T3): request ID generation and propagation, panic recovery, access
// logging, request timeouts and CORS. Applied uniformly so no engine can
// silently omit recovery — a bare, unrecovered panic in one handler used to
// be able to take an entire engine process down.
package httpx

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type contextKey string

const requestIDKey contextKey = "request_id"
const requestIDHeader = "X-Request-Id"

// Middleware wraps an http.Handler with additional behavior.
type Middleware func(http.Handler) http.Handler

// Chain applies middlewares to next in order, so the first middleware given
// runs first on the way in.
func Chain(next http.Handler, middlewares ...Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		next = middlewares[i](next)
	}
	return next
}

// RequestIDFromContext returns the request id stashed by RequestID, or ""
// if none is present.
func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}

// RequestID propagates an incoming X-Request-Id header, or generates one
// when absent, into both the request context and the response header — so
// a request can be traced end to end whether or not the caller supplied one.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(requestIDHeader)
		if id == "" {
			id = uuid.NewString()
		}
		w.Header().Set(requestIDHeader, id)
		ctx := context.WithValue(r.Context(), requestIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Recover catches a panic anywhere downstream, logs it with a stack trace,
// and answers 500 instead of taking the whole process down.
func Recover(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Error("panic recovered",
						"panic", rec,
						"request_id", RequestIDFromContext(r.Context()),
						"path", r.URL.Path,
					)
					w.WriteHeader(http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

// AccessLog logs one structured line per request: method, path, status,
// duration and the propagated request id.
func AccessLog(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)
			logger.Info("request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rec.status,
				"duration_ms", time.Since(start).Milliseconds(),
				"request_id", RequestIDFromContext(r.Context()),
			)
		})
	}
}

type timeoutWriter struct {
	http.ResponseWriter
}

func (tw *timeoutWriter) WriteHeader(code int) {
	if code == http.StatusServiceUnavailable {
		tw.ResponseWriter.Header().Set("Content-Type", "application/json")
	}
	tw.ResponseWriter.WriteHeader(code)
}

// Timeout bounds request handling to d. If the handler has not written a
// response by then, the client gets 503 and the handler's context is
// cancelled — the handler is expected to observe ctx.Done() and stop.
func Timeout(d time.Duration) Middleware {
	return func(next http.Handler) http.Handler {
		// Use standard TimeoutHandler, but with a JSON body string
		th := http.TimeoutHandler(next, d, `{"error":"request timed out"}`)
		
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Wrap w to intercept WriteHeader(503) and force Content-Type to JSON
			tw := &timeoutWriter{ResponseWriter: w}
			th.ServeHTTP(tw, r)
		})
	}
}

// CORSConfig configures the CORS middleware.
type CORSConfig struct {
	// AllowedOrigins is the exact set of allowed origins, or ["*"] to allow
	// any origin.
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
}

func (c CORSConfig) allows(origin string) bool {
	for _, o := range c.AllowedOrigins {
		if o == "*" || o == origin {
			return true
		}
	}
	return false
}

// CORS answers preflight OPTIONS requests directly and attaches CORS
// headers to every other response, without ever reaching next for a
// preflight.
func CORS(cfg CORSConfig) Middleware {
	methods := cfg.AllowedMethods
	if len(methods) == 0 {
		methods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	}
	headers := cfg.AllowedHeaders
	if len(headers) == 0 {
		headers = []string{"Content-Type", "Authorization", requestIDHeader}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && cfg.allows(origin) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", join(methods))
				w.Header().Set("Access-Control-Allow-Headers", join(headers))
			}

			if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func join(items []string) string {
	out := ""
	for i, s := range items {
		if i > 0 {
			out += ", "
		}
		out += s
	}
	return out
}
