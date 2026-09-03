import fs from "fs";
import https from "https";

export interface MeshTLSConfig {
  enabled: boolean;
  caFile?: string;
  certFile?: string;
  keyFile?: string;
  agent?: https.Agent;
}

let cachedMeshConfig: MeshTLSConfig | null = null;

/**
 * Returns the client mTLS configuration for service-to-service communication.
 */
export function getMeshTLSConfig(): MeshTLSConfig {
  if (cachedMeshConfig) return cachedMeshConfig;

  const certFile = process.env.AUTORIX_TLS_CLIENT_CERT_FILE || process.env.TLS_CLIENT_CERT_FILE;
  const keyFile = process.env.AUTORIX_TLS_CLIENT_KEY_FILE || process.env.TLS_CLIENT_KEY_FILE;
  const caFile = process.env.AUTORIX_TLS_CA_FILE || process.env.TLS_CA_FILE;
  const insecure = process.env.AUTORIX_TLS_INSECURE_SKIP_VERIFY === "true";

  if (!certFile || !keyFile) {
    cachedMeshConfig = { enabled: false };
    return cachedMeshConfig;
  }

  try {
    const cert = fs.readFileSync(certFile);
    const key = fs.readFileSync(keyFile);
    const ca = caFile && fs.existsSync(caFile) ? fs.readFileSync(caFile) : undefined;

    const agent = new https.Agent({
      cert,
      key,
      ca,
      rejectUnauthorized: !insecure,
    });

    cachedMeshConfig = {
      enabled: true,
      caFile,
      certFile,
      keyFile,
      agent,
    };
  } catch {
    cachedMeshConfig = { enabled: false };
  }

  return cachedMeshConfig;
}

/**
 * Resets the cached configuration (primarily for testing).
 */
export function resetMeshTLSConfig(): void {
  cachedMeshConfig = null;
}
