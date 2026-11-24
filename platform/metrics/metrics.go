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

	postgresPoolCollectorInstance = &postgresPoolCollector{
		totalDesc: prometheus.NewDesc(
			"autorix_postgres_pool_total_conns",
			"Total number of connections in the Postgres pool.",
			[]string{"engine"},
			nil,
		),
		idleDesc: prometheus.NewDesc(
			"autorix_postgres_pool_idle_conns",
			"Number of idle connections in the Postgres pool.",
			[]string{"engine"},
			nil,
		),
		acquiredDesc: prometheus.NewDesc(
			"autorix_postgres_pool_acquired_conns",
			"Number of currently acquired connections in the Postgres pool.",
			[]string{"engine"},
			nil,
		),
		maxDesc: prometheus.NewDesc(
			"autorix_postgres_pool_max_conns",
			"Maximum number of connections allowed in the Postgres pool.",
			[]string{"engine"},
			nil,
		),
	}

	registeredPoolsMu sync.RWMutex
	registeredPools   = make(map[string]poolStatProvider)
)

type poolStatSnapshot struct {
	total    int32
	idle     int32
	acquired int32
	max      int32
}

type poolStatProvider func() poolStatSnapshot

type postgresPoolCollector struct {
	totalDesc    *prometheus.Desc
	idleDesc     *prometheus.Desc
	acquiredDesc *prometheus.Desc
	maxDesc      *prometheus.Desc
}

func (c *postgresPoolCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.totalDesc
	ch <- c.idleDesc
	ch <- c.acquiredDesc
	ch <- c.maxDesc
}

func (c *postgresPoolCollector) Collect(ch chan<- prometheus.Metric) {
	registeredPoolsMu.RLock()
	snapshot := make(map[string]poolStatProvider, len(registeredPools))
	for engine, provider := range registeredPools {
		snapshot[engine] = provider
	}
	registeredPoolsMu.RUnlock()

	for engine, provider := range snapshot {
		if provider == nil {
			continue
		}
		stat := provider()
		ch <- prometheus.MustNewConstMetric(c.totalDesc, prometheus.GaugeValue, float64(stat.total), engine)
		ch <- prometheus.MustNewConstMetric(c.idleDesc, prometheus.GaugeValue, float64(stat.idle), engine)
		ch <- prometheus.MustNewConstMetric(c.acquiredDesc, prometheus.GaugeValue, float64(stat.acquired), engine)
		ch <- prometheus.MustNewConstMetric(c.maxDesc, prometheus.GaugeValue, float64(stat.max), engine)
	}
}

func init() {
	registerCollector(httpRequestsTotal)
	registerCollector(httpRequestDuration)
	registerCollector(grpcRequestsTotal)
	registerCollector(grpcRequestDuration)
	registerCollector(postgresPoolCollectorInstance)
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

func (s *statusRecorder) Unwrap() http.ResponseWriter {
	return s.ResponseWriter
}

// HTTPMiddleware returns a standard HTTP RED middleware measuring request counts and latencies.
// It uses the matched route pattern (r.Pattern) instead of the raw URL path to prevent
// unbounded Prometheus label cardinality attacks.
func HTTPMiddleware(engine string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)
			duration := time.Since(start).Seconds()

			path := r.Pattern
			if path == "" {
				path = "unmatched"
			}
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

// RegisterPostgresPool hooks a pgxpool.Pool's statistics to the live Prometheus collector.
func RegisterPostgresPool(pool *pgxpool.Pool, engine string) {
	if pool == nil {
		return
	}

	registerPoolStatProvider(engine, func() poolStatSnapshot {
		stat := pool.Stat()
		return poolStatSnapshot{
			total:    stat.TotalConns(),
			idle:     stat.IdleConns(),
			acquired: stat.AcquiredConns(),
			max:      stat.MaxConns(),
		}
	})
}

func registerPoolStatProvider(engine string, provider poolStatProvider) {
	registeredPoolsMu.Lock()
	registeredPools[engine] = provider
	registeredPoolsMu.Unlock()
}

// RegisterPoolStats hooks a pgxpool.Pool's statistics to Prometheus gauges with (engine, pool) ordering.
func RegisterPoolStats(engine string, pool *pgxpool.Pool) {
	RegisterPostgresPool(pool, engine)
}

// UpdatePostgresPoolStats is retained for backwards compatibility; pool statistics are
// sampled live on every Prometheus scrape via postgresPoolCollector.
func UpdatePostgresPoolStats(engine string) {
	registeredPoolsMu.RLock()
	provider, ok := registeredPools[engine]
	registeredPoolsMu.RUnlock()

	if !ok || provider == nil {
		return
	}
	_ = provider()
}
