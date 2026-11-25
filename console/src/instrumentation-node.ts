import dns from "dns";
import net from "net";

interface UndiciAgentCtor {
  new (options: Record<string, unknown>): unknown;
}

type LookupCallback = (
  err: NodeJS.ErrnoException | null,
  address: string | dns.LookupAddress[],
  family?: number,
) => void;

const DISPATCHER_SYMBOL = Symbol.for("undici.globalDispatcher.1");

function invokeDnsLookup(hostname: string, options: dns.LookupOptions, callback: LookupCallback): void {
  (dns.lookup as unknown as (host: string, opts: dns.LookupOptions, cb: LookupCallback) => void)(
    hostname,
    options,
    callback,
  );
}

/**
 * Resolve IPv4 addresses first on container bridges to avoid 10s Happy Eyeballs
 * stalls under Node 22 + Alpine musl while preserving automatic IPv6 fallback.
 */
export function ipv4FirstLookup(
  hostname: string,
  options: dns.LookupOptions | number | undefined,
  callback: LookupCallback,
): void {
  const baseOptions: dns.LookupOptions =
    typeof options === "number" ? { family: options } : options && typeof options === "object" ? options : {};

  if (baseOptions.family === 6) {
    invokeDnsLookup(hostname, baseOptions, callback);
    return;
  }

  const ipv4Options: dns.LookupOptions = { ...baseOptions, family: 4 };
  invokeDnsLookup(hostname, ipv4Options, (err, address, family) => {
    const hasAddress = !err && (!Array.isArray(address) || address.length > 0);
    if (hasAddress) {
      if (Array.isArray(address)) {
        callback(null, address);
      } else {
        callback(null, address, family);
      }
      return;
    }
    invokeDnsLookup(hostname, baseOptions, callback);
  });
}

export async function registerNodeNetworking(): Promise<boolean> {
  dns.setDefaultResultOrder("ipv4first");
  if (typeof net.setDefaultAutoSelectFamily === "function") {
    net.setDefaultAutoSelectFamily(false);
  }

  const globalStore = globalThis as unknown as Record<symbol, unknown>;

  // Prime Node's lazy built-in Undici dispatcher so we can construct a tuned
  // Agent with the exact same bundled class (zero external dependency).
  if (!globalStore[DISPATCHER_SYMBOL]) {
    await fetch("http://127.0.0.1:1", { signal: AbortSignal.timeout(25) }).catch(() => {});
  }

  const current = globalStore[DISPATCHER_SYMBOL] as { constructor?: UndiciAgentCtor } | undefined;
  const AgentCtor = current?.constructor;
  if (typeof AgentCtor !== "function") {
    return false;
  }

  globalStore[DISPATCHER_SYMBOL] = new AgentCtor({
    connections: 64,
    keepAliveTimeout: 5_000,
    keepAliveMaxTimeout: 15_000,
    connectTimeout: 3_000,
    connect: {
      autoSelectFamily: false,
      lookup: ipv4FirstLookup,
    },
  });

  console.info("[network] configured IPv4-first undici dispatcher");
  return true;
}
