import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { getMeshTLSConfig, resetMeshTLSConfig } from "./tls-mesh";
import fs from "fs";
import path from "path";
import os from "os";

describe("getMeshTLSConfig", () => {
  const originalEnv = process.env;

  beforeEach(() => {
    process.env = { ...originalEnv };
    resetMeshTLSConfig();
  });

  afterEach(() => {
    process.env = originalEnv;
    resetMeshTLSConfig();
  });

  it("returns enabled: false when TLS env vars are not set", () => {
    delete process.env.AUTORIX_TLS_CLIENT_CERT_FILE;
    delete process.env.AUTORIX_TLS_CLIENT_KEY_FILE;
    const config = getMeshTLSConfig();
    expect(config.enabled).toBe(false);
    expect(config.agent).toBeUndefined();
  });

  it("returns enabled: true and creates an HTTPS agent when cert and key exist", () => {
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "autorix-tls-test-"));
    const certPath = path.join(tmpDir, "test.crt");
    const keyPath = path.join(tmpDir, "test.key");

    fs.writeFileSync(certPath, "MOCK_CERT");
    fs.writeFileSync(keyPath, "MOCK_KEY");

    process.env.AUTORIX_TLS_CLIENT_CERT_FILE = certPath;
    process.env.AUTORIX_TLS_CLIENT_KEY_FILE = keyPath;

    const config = getMeshTLSConfig();
    expect(config.enabled).toBe(true);
    expect(config.agent).toBeDefined();

    fs.rmSync(tmpDir, { recursive: true, force: true });
  });
});
