import type { LogEntry, TraceDetail } from "../api/schemas/observability";

const MAX_LOGS = 300;
const MAX_TRACES = 100;

class TelemetryStore {
  private logs: LogEntry[] = [];
  private traces: Map<string, TraceDetail> = new Map();

  constructor() {
    this.seedInitialEvents();
  }

  private seedInitialEvents() {
    const now = Date.now();
    const engines = ["aegis", "ego", "janus", "nexus", "themis", "vulcan", "hermes", "argus"];
    
    engines.forEach((engine, idx) => {
      const ts = new Date(now - (engines.length - idx) * 30000).toISOString();
      const traceId = `tr-boot-${engine}`;
      
      this.recordLog({
        id: `log-boot-${engine}`,
        timestamp: ts,
        engine,
        instance_id: `${engine}-inst-1`,
        level: "info",
        message: `Engine ${engine.toUpperCase()} initialized: telemetry probes active on internal mesh`,
        request_id: `req-boot-${engine}`,
        correlation_id: `corr-boot-${engine}`,
        trace_id: traceId,
        attributes: { protocol: "http/grpc", status: "ready" },
      });

      this.traces.set(traceId, {
        trace_id: traceId,
        root_operation: `BOOT /health/ready [${engine}]`,
        root_engine: engine,
        total_duration_ms: 1.2 + idx * 0.3,
        timestamp: ts,
        status: "ok",
        spans: [
          {
            span_id: `sp-boot-${engine}`,
            trace_id: traceId,
            engine,
            operation: "healthcheck_probe",
            status: "ok",
            start_time_ms: 0,
            duration_ms: 1.2 + idx * 0.3,
            attributes: { endpoint: "/health/ready", response_code: 200 },
          },
        ],
      });
    });

    // Seed canonical multi-hop trace for distributed tracing waterfall
    const canonicalTraceId = "tr-7710a";
    this.traces.set(canonicalTraceId, {
      trace_id: canonicalTraceId,
      root_operation: "HTTP GET /api/v1/projects/alpha/deployments",
      root_engine: "aegis",
      total_duration_ms: 4.6,
      timestamp: new Date().toISOString(),
      status: "ok",
      spans: [
        {
          span_id: "sp-root",
          trace_id: canonicalTraceId,
          engine: "aegis",
          operation: "ingress_route_match",
          status: "ok",
          start_time_ms: 0,
          duration_ms: 4.6,
          attributes: { route: "/api/v1/projects/*", upstream: "http://deploy-engine:8080" },
        },
        {
          span_id: "sp-vulcan",
          parent_span_id: "sp-root",
          trace_id: canonicalTraceId,
          engine: "vulcan",
          operation: "verify_api_key",
          status: "ok",
          start_time_ms: 0.6,
          duration_ms: 0.8,
          attributes: { prefix: "av_live_78ab", scopes: ["read", "deploy"] },
        },
        {
          span_id: "sp-nexus",
          parent_span_id: "sp-root",
          trace_id: canonicalTraceId,
          engine: "nexus",
          operation: "check_relation",
          status: "ok",
          start_time_ms: 1.6,
          duration_ms: 1.4,
          attributes: { subject: "user:operator", relation: "deployer" },
        },
        {
          span_id: "sp-themis",
          parent_span_id: "sp-root",
          trace_id: canonicalTraceId,
          engine: "themis",
          operation: "evaluate_policy",
          status: "ok",
          start_time_ms: 3.2,
          duration_ms: 0.6,
          attributes: { policy: "allow-deployments-in-region" },
        },
        {
          span_id: "sp-upstream",
          parent_span_id: "sp-root",
          trace_id: canonicalTraceId,
          engine: "aegis",
          operation: "upstream_proxy_roundtrip",
          status: "ok",
          start_time_ms: 3.9,
          duration_ms: 0.7,
          attributes: { http_status: 200 },
        },
      ],
    });
  }

  public recordLog(entry: LogEntry): void {
    this.logs.unshift(entry);
    if (this.logs.length > MAX_LOGS) {
      this.logs.pop();
    }
  }

  public recordRequest(params: {
    service: string;
    path: string;
    method: string;
    status: number;
    durationMs: number;
    requestId: string;
    correlationId: string;
    error?: string;
  }): void {
    const { service, path, method, status, durationMs, requestId, correlationId, error } = params;
    const isError = status >= 500;
    const isWarn = status >= 400 && status < 500;
    const level: "debug" | "info" | "warn" | "error" = isError ? "error" : isWarn ? "warn" : "info";
    const ts = new Date().toISOString();

    const logEntry: LogEntry = {
      id: `log-${Date.now()}-${requestId.slice(0, 6)}`,
      timestamp: ts,
      engine: service,
      instance_id: `${service}-inst-1`,
      level,
      message: `${method} ${path} -> ${status} (${durationMs.toFixed(1)}ms)${error ? ` - ${error}` : ""}`,
      request_id: requestId,
      correlation_id: correlationId,
      trace_id: correlationId,
      attributes: {
        method,
        path,
        status,
        duration_ms: durationMs,
        ...(error ? { error } : {}),
      },
    };
    this.recordLog(logEntry);

    const traceId = correlationId;
    const spanId = `sp-${requestId.slice(0, 8)}`;
    const spanStatus: "ok" | "error" = isError ? "error" : "ok";

    const existingTrace = this.traces.get(traceId);
    if (existingTrace) {
      existingTrace.spans.push({
        span_id: spanId,
        parent_span_id: existingTrace.spans[0]?.span_id,
        trace_id: traceId,
        engine: service,
        operation: `${method} ${path}`,
        status: spanStatus,
        start_time_ms: 0.5,
        duration_ms: durationMs,
        attributes: { status, path, method },
      });
      existingTrace.total_duration_ms = Math.max(existingTrace.total_duration_ms, durationMs);
    } else {
      const newTrace: TraceDetail = {
        trace_id: traceId,
        root_operation: `${method} ${path}`,
        root_engine: service,
        total_duration_ms: durationMs,
        timestamp: ts,
        status: spanStatus,
        spans: [
          {
            span_id: spanId,
            trace_id: traceId,
            engine: service,
            operation: `${method} ${path}`,
            status: spanStatus,
            start_time_ms: 0,
            duration_ms: durationMs,
            attributes: { status, path, method },
          },
        ],
      };
      this.traces.set(traceId, newTrace);
      if (this.traces.size > MAX_TRACES) {
        const oldestKey = this.traces.keys().next().value;
        if (oldestKey) this.traces.delete(oldestKey);
      }
    }
  }

  public getLogs(filter?: { engine?: string; level?: string; limit?: number }): LogEntry[] {
    let result = [...this.logs];
    if (filter?.engine && filter.engine !== "all") {
      result = result.filter((l) => l.engine.toLowerCase() === filter.engine?.toLowerCase());
    }
    if (filter?.level && filter.level !== "all") {
      result = result.filter((l) => l.level === filter.level);
    }
    const limit = filter?.limit || 50;
    return result.slice(0, limit);
  }

  public getTraces(limit = 20): TraceDetail[] {
    const list = Array.from(this.traces.values());
    list.sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime());
    return list.slice(0, limit);
  }

  public getTraceDetail(traceId: string): TraceDetail | undefined {
    return this.traces.get(traceId);
  }
}

// Global singleton instance preserved across Next.js invocations
const globalForTelemetry = globalThis as unknown as { telemetryStore?: TelemetryStore };
export const telemetryStore = globalForTelemetry.telemetryStore ?? new TelemetryStore();
if (process.env.NODE_ENV !== "production") {
  globalForTelemetry.telemetryStore = telemetryStore;
}
