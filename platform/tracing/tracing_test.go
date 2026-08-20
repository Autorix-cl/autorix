package tracing

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestHTTPMiddleware_GeneratesNewIDsWhenMissing(t *testing.T) {
	var capturedReqID, capturedCorrID string

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedReqID = RequestIDFromContext(r.Context())
		capturedCorrID = CorrelationIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	mw := HTTPMiddleware()(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	if capturedReqID == "" {
		t.Error("expected generated request_id, got empty")
	}
	if capturedCorrID == "" {
		t.Error("expected generated correlation_id, got empty")
	}
	if capturedReqID != capturedCorrID {
		t.Errorf("expected correlation_id to match generated request_id, got %q vs %q", capturedCorrID, capturedReqID)
	}

	if rec.Header().Get(HeaderRequestID) != capturedReqID {
		t.Errorf("expected X-Request-ID response header %q, got %q", capturedReqID, rec.Header().Get(HeaderRequestID))
	}
	if rec.Header().Get(HeaderCorrelationID) != capturedCorrID {
		t.Errorf("expected X-Correlation-ID response header %q, got %q", capturedCorrID, rec.Header().Get(HeaderCorrelationID))
	}
}

func TestHTTPMiddleware_PropagatesExistingIDs(t *testing.T) {
	var capturedReqID, capturedCorrID string

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedReqID = RequestIDFromContext(r.Context())
		capturedCorrID = CorrelationIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	mw := Middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(HeaderRequestID, "req-12345")
	req.Header.Set(HeaderCorrelationID, "corr-67890")

	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	if capturedReqID != "req-12345" {
		t.Errorf("expected request_id 'req-12345', got %q", capturedReqID)
	}
	if capturedCorrID != "corr-67890" {
		t.Errorf("expected correlation_id 'corr-67890', got %q", capturedCorrID)
	}

	if rec.Header().Get(HeaderRequestID) != "req-12345" {
		t.Errorf("expected response X-Request-ID 'req-12345', got %q", rec.Header().Get(HeaderRequestID))
	}
	if rec.Header().Get(HeaderCorrelationID) != "corr-67890" {
		t.Errorf("expected response X-Correlation-ID 'corr-67890', got %q", rec.Header().Get(HeaderCorrelationID))
	}
}

func TestHTTPMiddleware_AlternateCaseHeaders(t *testing.T) {
	var capturedReqID, capturedCorrID string

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedReqID = RequestIDFromContext(r.Context())
		capturedCorrID = CorrelationIDFromContext(r.Context())
	})

	mw := HTTPMiddleware()(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Request-Id", "req-alt")
	req.Header.Set("X-Correlation-Id", "corr-alt")

	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	if capturedReqID != "req-alt" {
		t.Errorf("expected request_id 'req-alt', got %q", capturedReqID)
	}
	if capturedCorrID != "corr-alt" {
		t.Errorf("expected correlation_id 'corr-alt', got %q", capturedCorrID)
	}
}

func TestUnaryServerInterceptor_ExtractsAndPropagates(t *testing.T) {
	interceptor := UnaryServerInterceptor()

	// Test 1: With existing metadata
	incomingMD := metadata.Pairs(
		MDRequestID, "grpc-req-1",
		MDCorrelationID, "grpc-corr-1",
	)
	ctx := metadata.NewIncomingContext(context.Background(), incomingMD)

	var ctxReqID, ctxCorrID string
	handler := func(ctx context.Context, req any) (any, error) {
		ctxReqID = RequestIDFromContext(ctx)
		ctxCorrID = CorrelationIDFromContext(ctx)
		return "ok", nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/service/method"}
	_, err := interceptor(ctx, "req", info, handler)
	if err != nil {
		t.Fatalf("interceptor returned error: %v", err)
	}

	if ctxReqID != "grpc-req-1" {
		t.Errorf("expected reqID 'grpc-req-1', got %q", ctxReqID)
	}
	if ctxCorrID != "grpc-corr-1" {
		t.Errorf("expected corrID 'grpc-corr-1', got %q", ctxCorrID)
	}

	// Test 2: With empty metadata -> should generate new UUIDs
	emptyCtx := context.Background()
	_, err = interceptor(emptyCtx, "req", info, handler)
	if err != nil {
		t.Fatalf("interceptor returned error: %v", err)
	}

	if ctxReqID == "" {
		t.Error("expected generated reqID, got empty")
	}
	if ctxCorrID != ctxReqID {
		t.Errorf("expected corrID to match reqID, got %q vs %q", ctxCorrID, ctxReqID)
	}
}

func TestUnaryClientInterceptor_InjectsMetadata(t *testing.T) {
	clientInterceptor := UnaryClientInterceptor()

	ctx := ContextWithIDs(context.Background(), "cli-req-123", "cli-corr-456")

	invoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			t.Fatal("expected outgoing metadata in context")
		}

		reqVals := md.Get(MDRequestID)
		if len(reqVals) == 0 || reqVals[0] != "cli-req-123" {
			t.Errorf("expected outgoing x-request-id 'cli-req-123', got %v", reqVals)
		}

		corrVals := md.Get(MDCorrelationID)
		if len(corrVals) == 0 || corrVals[0] != "cli-corr-456" {
			t.Errorf("expected outgoing x-correlation-id 'cli-corr-456', got %v", corrVals)
		}
		return nil
	}

	err := clientInterceptor(ctx, "/service/method", nil, nil, nil, invoker)
	if err != nil {
		t.Fatalf("client interceptor error: %v", err)
	}
}

func TestStructuredLogIntegration_LogHandler(t *testing.T) {
	buf := &bytes.Buffer{}
	baseHandler := slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	logHandler := NewLogHandler(baseHandler)
	logger := slog.New(logHandler)

	ctx := ContextWithIDs(context.Background(), "req-log-789", "corr-log-789")

	logger.InfoContext(ctx, "processing order", "order_id", 12345)

	var logEntry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("failed to unmarshal log output: %v, raw: %s", err, buf.String())
	}

	if logEntry["msg"] != "processing order" {
		t.Errorf("expected msg 'processing order', got %v", logEntry["msg"])
	}
	if logEntry["request_id"] != "req-log-789" {
		t.Errorf("expected request_id 'req-log-789', got %v", logEntry["request_id"])
	}
	if logEntry["correlation_id"] != "corr-log-789" {
		t.Errorf("expected correlation_id 'corr-log-789', got %v", logEntry["correlation_id"])
	}
	if logEntry["order_id"] != float64(12345) {
		t.Errorf("expected order_id 12345, got %v", logEntry["order_id"])
	}
}

func TestStructuredLogIntegration_WithOTelSpan(t *testing.T) {
	buf := &bytes.Buffer{}
	baseHandler := slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	logHandler := NewLogHandler(baseHandler)
	logger := slog.New(logHandler)

	// Create a valid OTel span context
	traceID, _ := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	spanID, _ := trace.SpanIDFromHex("00f067aa0ba902b7")
	spanContext := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})

	ctx := trace.ContextWithSpanContext(context.Background(), spanContext)
	ctx = ContextWithIDs(ctx, "req-otel-1", "corr-otel-1")

	logger.InfoContext(ctx, "tracing span event")

	var logEntry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("failed to parse log JSON: %v", err)
	}

	if logEntry["trace_id"] != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Errorf("expected trace_id '4bf92f3577b34da6a3ce929d0e0e4736', got %v", logEntry["trace_id"])
	}
	if logEntry["span_id"] != "00f067aa0ba902b7" {
		t.Errorf("expected span_id '00f067aa0ba902b7', got %v", logEntry["span_id"])
	}
}

func TestLoggerWithContext(t *testing.T) {
	buf := &bytes.Buffer{}
	baseLogger := slog.New(slog.NewJSONHandler(buf, nil))

	ctx := ContextWithIDs(context.Background(), "req-bound", "corr-bound")
	boundLogger := LoggerWithContext(baseLogger, ctx)

	boundLogger.Info("bound logger test")

	var logEntry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("failed to parse log: %v", err)
	}

	if logEntry["request_id"] != "req-bound" || logEntry["correlation_id"] != "corr-bound" {
		t.Errorf("expected bound request_id and correlation_id, got %v", logEntry)
	}
}
