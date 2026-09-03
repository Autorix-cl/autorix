import { describe, it, expect, vi, beforeEach } from "vitest";
import { POST as runProbe } from "./probe/route";
import { GET as getDrift } from "./drift/route";
import { GET as getMigrations } from "./migrations/route";
import { GET as getTimeline } from "./timeline/route";
import { POST as exportBundle } from "./bundle/route";
import { NextRequest } from "next/server";

describe("Diagnostics BFF Routes", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("POST /api/diagnostics/probe runs a live multi-stage connectivity probe", async () => {
    const req = new NextRequest("http://localhost/api/diagnostics/probe", {
      method: "POST",
      body: JSON.stringify({ source_engine: "aegis", target_engine: "nexus" }),
    });
    const res = await runProbe(req);
    expect(res.status).toBe(200);
    const data = await res.json();
    expect(data.source_engine).toBe("aegis");
    expect(data.target_engine).toBe("nexus");
    expect(data.dns_status).toBe("healthy");
    expect(data.tcp_status).toBe("healthy");
    expect(data.http_code).toBe(200);
    expect(data.overall_status).toBe("healthy");
  });

  it("GET /api/diagnostics/drift returns configuration drift findings", async () => {
    const res = await getDrift();
    expect(res.status).toBe(200);
    const data = await res.json();
    expect(Array.isArray(data)).toBe(true);
    expect(data.length).toBeGreaterThan(0);
    expect(data[0].parameter).toBeTruthy();
  });

  it("GET /api/diagnostics/migrations returns database migration status across engines", async () => {
    const res = await getMigrations();
    expect(res.status).toBe(200);
    const data = await res.json();
    expect(Array.isArray(data)).toBe(true);
    expect(data.length).toBeGreaterThanOrEqual(7);
    expect(data.every((m: { up_to_date: boolean }) => m.up_to_date)).toBe(true);
  });

  it("GET /api/diagnostics/timeline returns change correlation events", async () => {
    const res = await getTimeline();
    expect(res.status).toBe(200);
    const data = await res.json();
    expect(Array.isArray(data)).toBe(true);
    expect(data.some((e: { correlated_error_spike: boolean }) => e.correlated_error_spike)).toBe(true);
  });

  it("POST /api/diagnostics/bundle generates a redacted diagnostic archive", async () => {
    const res = await exportBundle();
    expect(res.status).toBe(201);
    const data = await res.json();
    expect(data.bundle_id).toBeTruthy();
    expect(data.redacted).toBe(true);
    expect(data.download_url).toContain(".tar.gz");
  });
});
