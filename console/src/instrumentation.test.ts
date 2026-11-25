import dns from "dns";
import { afterEach, describe, expect, it, vi } from "vitest";
import { register } from "./instrumentation";
import { ipv4FirstLookup, registerNodeNetworking } from "./instrumentation-node";

const DISPATCHER_SYMBOL = Symbol.for("undici.globalDispatcher.1");

describe("instrumentation networking", () => {
  const originalRuntime = process.env.NEXT_RUNTIME;
  const globalStore = globalThis as unknown as Record<symbol, unknown>;
  const originalDispatcher = globalStore[DISPATCHER_SYMBOL];

  afterEach(() => {
    vi.restoreAllMocks();
    if (originalRuntime === undefined) {
      delete process.env.NEXT_RUNTIME;
    } else {
      process.env.NEXT_RUNTIME = originalRuntime;
    }
    if (originalDispatcher === undefined) {
      delete globalStore[DISPATCHER_SYMBOL];
    } else {
      globalStore[DISPATCHER_SYMBOL] = originalDispatcher;
    }
  });

  it("prefers IPv4 in ipv4FirstLookup and returns the IPv4 result", () => {
    const spy = vi.spyOn(dns, "lookup").mockImplementation(((_host: string, opts: unknown, cb: unknown) => {
      const callback = cb as (err: null, address: string, family: number) => void;
      expect(opts).toEqual({ family: 4 });
      callback(null, "172.18.0.5", 4);
    }) as typeof dns.lookup);

    const done = vi.fn();
    ipv4FirstLookup("argus", {}, done);

    expect(spy).toHaveBeenCalledTimes(1);
    expect(done).toHaveBeenCalledWith(null, "172.18.0.5", 4);
  });

  it("falls back to default lookup when IPv4 resolution fails", () => {
    const spy = vi.spyOn(dns, "lookup").mockImplementation(((_host: string, opts: unknown, cb: unknown) => {
      const options = opts as dns.LookupOptions;
      const callback = cb as (err: NodeJS.ErrnoException | null, address?: string, family?: number) => void;
      if (options.family === 4) {
        callback(new Error("ENOTFOUND") as NodeJS.ErrnoException);
        return;
      }
      callback(null, "::1", 6);
    }) as typeof dns.lookup);

    const done = vi.fn();
    ipv4FirstLookup("ipv6-only.local", {}, done);

    expect(spy).toHaveBeenCalledTimes(2);
    expect(done).toHaveBeenCalledWith(null, "::1", 6);
  });

  it("respects explicit IPv6 requests without forcing IPv4", () => {
    const spy = vi.spyOn(dns, "lookup").mockImplementation(((_host: string, opts: unknown, cb: unknown) => {
      const callback = cb as (err: null, address: string, family: number) => void;
      expect(opts).toEqual({ family: 6 });
      callback(null, "::1", 6);
    }) as typeof dns.lookup);

    const done = vi.fn();
    ipv4FirstLookup("localhost", { family: 6 }, done);

    expect(spy).toHaveBeenCalledTimes(1);
    expect(done).toHaveBeenCalledWith(null, "::1", 6);
  });

  it("replaces the global undici dispatcher when running in nodejs runtime", async () => {
    class MockAgent {
      options: Record<string, unknown>;
      constructor(options: Record<string, unknown>) {
        this.options = options;
      }
    }

    globalStore[DISPATCHER_SYMBOL] = new MockAgent({});
    const infoSpy = vi.spyOn(console, "info").mockImplementation(() => {});

    const configured = await registerNodeNetworking();
    expect(configured).toBe(true);
    expect(infoSpy).toHaveBeenCalledWith("[network] configured IPv4-first undici dispatcher");

    const updated = globalStore[DISPATCHER_SYMBOL] as MockAgent;
    expect(updated).toBeInstanceOf(MockAgent);
    expect(updated.options.connectTimeout).toBe(3_000);
    expect((updated.options.connect as Record<string, unknown>).autoSelectFamily).toBe(false);
    expect((updated.options.connect as Record<string, unknown>).lookup).toBe(ipv4FirstLookup);
  });

  it("skips Node networking setup when NEXT_RUNTIME is not nodejs", async () => {
    process.env.NEXT_RUNTIME = "edge";
    const infoSpy = vi.spyOn(console, "info").mockImplementation(() => {});

    await register();

    expect(infoSpy).not.toHaveBeenCalled();
  });
});
