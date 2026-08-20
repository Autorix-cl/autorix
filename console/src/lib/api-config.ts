export const BACKEND_URLS = {
  ego: process.env.EGO_INTERNAL_URL || "http://ego:4433",
  janus: process.env.JANUS_INTERNAL_URL || "http://janus:4444",
  vulcan: process.env.VULCAN_INTERNAL_URL || "http://vulcan:4466",
  hermes: process.env.HERMES_INTERNAL_URL || "http://hermes:4477",
  // Aegis's proxy pipeline (:4455) is a catch-all reverse proxy, not an API
  // the console can call. This points at its admin REST API (:4456) instead,
  // which the console uses to manage routing rules.
  aegis: process.env.AEGIS_INTERNAL_URL || "http://aegis:4456",
  // Nexus's canonical transport is gRPC on :50051, but that can't be reached
  // with a plain fetch(). This points at its REST admin API (:8080) instead,
  // which the console uses to manage relation tuples and run Check.
  nexus: process.env.NEXUS_INTERNAL_URL || "http://nexus:8080",
  themis: process.env.THEMIS_INTERNAL_URL || "http://themis:4488",
  argus: process.env.ARGUS_INTERNAL_URL || "http://argus:4400",
};

export function getServiceUrl(service: keyof typeof BACKEND_URLS): string {
  // If running on server inside Docker container
  if (typeof window === "undefined") {
    return BACKEND_URLS[service] || `http://${service}:4433`;
  }
  // If running on client browser
  const publicPorts: Record<string, number> = {
    ego: 4433,
    janus: 4444,
    vulcan: 4466,
    hermes: 4477,
    aegis: 4456,
    nexus: 8080,
    themis: 4488,
    argus: 4400,
  };
  return `http://localhost:${publicPorts[service] || 3000}`;
}
