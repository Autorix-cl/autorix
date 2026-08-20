// Package tracing provides Request ID and Correlation ID propagation across HTTP and gRPC,
// OpenTelemetry trace context propagation, and structured logging integration (P7-S2-T1, P7-S2-T2).
package tracing

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	// HeaderRequestID is the standard HTTP header for request identification.
	HeaderRequestID = "X-Request-ID"
	// HeaderCorrelationID is the standard HTTP header for distributed correlation identification.
	HeaderCorrelationID = "X-Correlation-ID"

	// HeaderRequestIDAlt is the alternate mixed-case request ID header.
	HeaderRequestIDAlt = "X-Request-Id"
	// HeaderCorrelationIDAlt is the alternate mixed-case correlation ID header.
	HeaderCorrelationIDAlt = "X-Correlation-Id"

	// MDRequestID is the gRPC metadata key for request identification.
	MDRequestID = "x-request-id"
	// MDCorrelationID is the gRPC metadata key for correlation identification.
	MDCorrelationID = "x-correlation-id"
)

type contextKey string

const (
	requestIDKey     contextKey = "request_id"
	correlationIDKey contextKey = "correlation_id"
)

// RequestIDFromContext extracts the request ID from ctx, returning "" if not found.
// Checks both typed and string context keys for compatibility.
func RequestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if id, ok := ctx.Value(requestIDKey).(string); ok && id != "" {
		return id
	}
	if id, ok := ctx.Value("request_id").(string); ok && id != "" {
		return id
	}
	return ""
}

// CorrelationIDFromContext extracts the correlation ID from ctx, returning "" if not found.
// Checks both typed and string context keys for compatibility.
func CorrelationIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if id, ok := ctx.Value(correlationIDKey).(string); ok && id != "" {
		return id
	}
	if id, ok := ctx.Value("correlation_id").(string); ok && id != "" {
		return id
	}
	return ""
}

// ContextWithIDs returns a new context containing the given request ID and correlation ID.
func ContextWithIDs(ctx context.Context, requestID, correlationID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx = context.WithValue(ctx, requestIDKey, requestID)
	ctx = context.WithValue(ctx, correlationIDKey, correlationID)
	return ctx
}

// ContextWithRequestID returns a new context with the given request ID.
func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, requestIDKey, requestID)
}

// ContextWithCorrelationID returns a new context with the given correlation ID.
func ContextWithCorrelationID(ctx context.Context, correlationID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, correlationIDKey, correlationID)
}

// ExtractTraceContext extracts the OpenTelemetry trace ID and span ID if active in ctx.
func ExtractTraceContext(ctx context.Context) (traceID, spanID string) {
	if ctx == nil {
		return "", ""
	}
	spanCtx := trace.SpanContextFromContext(ctx)
	if !spanCtx.IsValid() {
		return "", ""
	}
	return spanCtx.TraceID().String(), spanCtx.SpanID().String()
}

// HTTPMiddleware extracts or generates X-Request-ID and X-Correlation-ID headers,
// sets them into request context and response headers, and passes down to next handler.
func HTTPMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID := r.Header.Get(HeaderRequestID)
			if reqID == "" {
				reqID = r.Header.Get(HeaderRequestIDAlt)
			}
			if reqID == "" {
				reqID = uuid.NewString()
			}

			corrID := r.Header.Get(HeaderCorrelationID)
			if corrID == "" {
				corrID = r.Header.Get(HeaderCorrelationIDAlt)
			}
			if corrID == "" {
				corrID = reqID
			}

			w.Header().Set(HeaderRequestID, reqID)
			w.Header().Set(HeaderCorrelationID, corrID)
			w.Header().Set(HeaderRequestIDAlt, reqID)

			ctx := ContextWithIDs(r.Context(), reqID, corrID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Middleware is an alias for HTTPMiddleware()(next) for direct middleware chaining.
func Middleware(next http.Handler) http.Handler {
	return HTTPMiddleware()(next)
}

// UnaryServerInterceptor returns a gRPC server interceptor that extracts or generates
// request and correlation IDs from gRPC incoming metadata, injects them into context,
// and sets them on gRPC response headers.
func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		var reqID, corrID string
		if ok {
			if vals := md.Get(MDRequestID); len(vals) > 0 {
				reqID = vals[0]
			}
			if vals := md.Get(MDCorrelationID); len(vals) > 0 {
				corrID = vals[0]
			}
		}

		if reqID == "" {
			reqID = uuid.NewString()
		}
		if corrID == "" {
			corrID = reqID
		}

		ctx = ContextWithIDs(ctx, reqID, corrID)
		_ = grpc.SetHeader(ctx, metadata.Pairs(
			MDRequestID, reqID,
			MDCorrelationID, corrID,
		))

		return handler(ctx, req)
	}
}

// UnaryClientInterceptor returns a gRPC client interceptor that propagates request ID
// and correlation ID from context to outgoing gRPC metadata.
func UnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		reqID := RequestIDFromContext(ctx)
		corrID := CorrelationIDFromContext(ctx)

		if reqID != "" || corrID != "" {
			var pairs []string
			if reqID != "" {
				pairs = append(pairs, MDRequestID, reqID)
			}
			if corrID != "" {
				pairs = append(pairs, MDCorrelationID, corrID)
			}
			ctx = metadata.AppendToOutgoingContext(ctx, pairs...)
		}

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// LogHandler is an slog.Handler middleware that enriches log records with
// request_id, correlation_id, trace_id, and span_id from context.
type LogHandler struct {
	next slog.Handler
}

// NewLogHandler wraps an existing slog.Handler to inject tracing and request IDs.
func NewLogHandler(next slog.Handler) *LogHandler {
	return &LogHandler{next: next}
}

// Enabled reports whether the handler handles records at the given level.
func (h *LogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

// Handle enriches the slog.Record with context-derived attributes (request_id, correlation_id, trace_id, span_id).
func (h *LogHandler) Handle(ctx context.Context, r slog.Record) error {
	if ctx != nil {
		if reqID := RequestIDFromContext(ctx); reqID != "" {
			r.AddAttrs(slog.String("request_id", reqID))
		}
		if corrID := CorrelationIDFromContext(ctx); corrID != "" {
			r.AddAttrs(slog.String("correlation_id", corrID))
		}

		traceID, spanID := ExtractTraceContext(ctx)
		if traceID != "" {
			r.AddAttrs(
				slog.String("trace_id", traceID),
				slog.String("span_id", spanID),
			)
		}
	}
	return h.next.Handle(ctx, r)
}

// WithAttrs returns a new LogHandler whose attributes include the given attributes.
func (h *LogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &LogHandler{next: h.next.WithAttrs(attrs)}
}

// WithGroup returns a new LogHandler with the given group name.
func (h *LogHandler) WithGroup(name string) slog.Handler {
	return &LogHandler{next: h.next.WithGroup(name)}
}

// LoggerWithContext returns a new slog.Logger with request_id and correlation_id
// bound as attributes from the context, or the base logger if none present.
func LoggerWithContext(logger *slog.Logger, ctx context.Context) *slog.Logger {
	if logger == nil {
		logger = slog.Default()
	}
	if ctx == nil {
		return logger
	}
	var attrs []any
	if reqID := RequestIDFromContext(ctx); reqID != "" {
		attrs = append(attrs, "request_id", reqID)
	}
	if corrID := CorrelationIDFromContext(ctx); corrID != "" {
		attrs = append(attrs, "correlation_id", corrID)
	}
	if len(attrs) == 0 {
		return logger
	}
	return logger.With(attrs...)
}

// WithRequestID is an alias for LoggerWithContext for compatibility.
func WithRequestID(logger *slog.Logger, ctx context.Context) *slog.Logger {
	return LoggerWithContext(logger, ctx)
}
