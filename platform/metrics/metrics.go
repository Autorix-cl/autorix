// Package metrics provides shared Prometheus metrics, HTTP RED middleware,
// gRPC interceptors, and Postgres pool stats collectors across Autorix engines.
package metrics

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "autorix_http_requests_total",
			Help: "Total number of HTTP requests processed, partitioned by engine, method, path, and status code.",
		},
		[]string{"engine", "method", "path", "status"},
	)
	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "autorix_http_request_duration_seconds",
			Help:    "HTTP request latency in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"engine", "method", "path"},
	)

	grpcRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "autorix_grpc_requests_total",
			Help: "Total number of gRPC unary RPCs processed, partitioned by engine, method, and status code.",
		},
		[]string{"engine", "method", "code"},
	)
	grpcRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "autorix_grpc_request_duration_seconds",
			Help:    "gRPC unary RPC latency in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"engine", "method"},
	)

	postgresPoolTotalConns = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "autorix_postgres_pool_total_conns",
			Help: "Total number of connections in the Postgres pool.",
		},
		[]string{"engine"},
	)
	postgresPoolIdleConns = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "autorix_postgres_pool_idle_conns",
			Help: "Number of idle connections in the Postgres pool.",
		},
		[]string{"engine"},
	)
	postgresPoolAcquiredConns = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "autorix_postgres_pool_acquired_conns",
			Help: "Number of currently acquired connections in the Postgres pool.",
		},
		[]string{"engine"},
	)
	postgresPoolMaxConns = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "autorix_postgres_pool_max_conns",
			Help: "Maximum number of connections allowed in the Postgres pool.",
		},
		[]string{"engine"},
	)

	registeredPoolsMu sync.Mutex
	registeredPools   = make(map[string]*pgxpool.Pool)
)

func init() {
	registerCollector(httpRequestsTotal)
	registerCollector(httpRequestDuration)
	registerCollector(grpcRequestsTotal)
	registerCollector(grpcRequestDuration)
	registerCollector(postgresPoolTotalConns)
	registerCollector(postgresPoolIdleConns)
	registerCollector(postgresPoolAcquiredConns)
	registerCollector(postgresPoolMaxConns)
}

func registerCollector(c prometheus.Collector) {
	if err := prometheus.Register(c); err != nil {
		if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {
			// Ignore already registered collectors in test runs or re-inits
		}
	}
}

// Handler returns an http.Handler that exposes Prometheus metrics on the default registry.
func Handler() http.Handler {
	return promhttp.Handler()
}

type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (s *statusRecorder) WriteHeader(code int) {
	if !s.wroteHeader {
		s.status = code
		s.wroteHeader = true
	}
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if !s.wroteHeader {
		s.status = http.StatusOK
		s.wroteHeader = true
	}
	return s.ResponseWriter.Write(b)
}

func (s *statusRecorder) Flush() {
	if flusher, ok := s.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// HTTPMiddleware returns a standard HTTP RED middleware measuring request counts and latencies.
func HTTPMiddleware(engine string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)
			duration := time.Since(start).Seconds()

			path := r.URL.Path
			httpRequestsTotal.WithLabelValues(engine, r.Method, path, strconv.Itoa(rec.status)).Inc()
			httpRequestDuration.WithLabelValues(engine, r.Method, path).Observe(duration)
		})
	}
}

// UnaryServerInterceptor returns a gRPC unary server interceptor measuring RPC counts and latencies.
func UnaryServerInterceptor(engine string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		duration := time.Since(start).Seconds()
		code := status.Code(err).String()

		grpcRequestsTotal.WithLabelValues(engine, info.FullMethod, code).Inc()
		grpcRequestDuration.WithLabelValues(engine, info.FullMethod).Observe(duration)

		return resp, err
	}
}

// StreamServerInterceptor returns a gRPC stream server interceptor.
func StreamServerInterceptor(engine string) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()
		err := handler(srv, ss)
		duration := time.Since(start).Seconds()
		code := status.Code(err).String()

		grpcRequestsTotal.WithLabelValues(engine, info.FullMethod, code).Inc()
		grpcRequestDuration.WithLabelValues(engine, info.FullMethod).Observe(duration)

		return err
	}
}

// RegisterPostgresPool hooks a pgxpool.Pool's statistics to Prometheus gauges.
func RegisterPostgresPool(pool *pgxpool.Pool, engine string) {
	if pool == nil {
		return
	}

	registeredPoolsMu.Lock()
	registeredPools[engine] = pool
	registeredPoolsMu.Unlock()

	// Perform an initial sampling
	UpdatePostgresPoolStats(engine)
}

// RegisterPoolStats hooks a pgxpool.Pool's statistics to Prometheus gauges with (engine, pool) ordering.
func RegisterPoolStats(engine string, pool *pgxpool.Pool) {
	RegisterPostgresPool(pool, engine)
}

// UpdatePostgresPoolStats refreshes gauges for a registered pool.
func UpdatePostgresPoolStats(engine string) {
	registeredPoolsMu.Lock()
	pool, ok := registeredPools[engine]
	registeredPoolsMu.Unlock()

	if !ok || pool == nil {
		return
	}

	stat := pool.Stat()
	postgresPoolTotalConns.WithLabelValues(engine).Set(float64(stat.TotalConns()))
	postgresPoolIdleConns.WithLabelValues(engine).Set(float64(stat.IdleConns()))
	postgresPoolAcquiredConns.WithLabelValues(engine).Set(float64(stat.AcquiredConns()))
	postgresPoolMaxConns.WithLabelValues(engine).Set(float64(stat.MaxConns()))
}
