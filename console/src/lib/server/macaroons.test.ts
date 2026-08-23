import { describe, it, expect } from "vitest";
import { generateRootSecret, createMacaroon, addCaveat, verifyMacaroon, serializeMacaroon, deserializeMacaroon } from "./macaroons";

describe("Macaroons Cryptography", () => {
  it("should generate a valid root secret", () => {
    const secret = generateRootSecret();
    expect(secret).toBeDefined();
    expect(secret.length).toBeGreaterThan(32);
  });

  it("should create a macaroon and verify it successfully", () => {
    const rootSecret = generateRootSecret();
    const keyId = "key_123";
    
    const macaroon = createMacaroon(keyId, rootSecret);
    
    expect(macaroon.id).toBe(keyId);
    expect(macaroon.caveats).toHaveLength(0);
    expect(verifyMacaroon(macaroon, rootSecret)).toBe(true);
  });

  it("should fail verification if root secret is wrong", () => {
    const rootSecret1 = generateRootSecret();
    const rootSecret2 = generateRootSecret();
    
    const macaroon = createMacaroon("key_123", rootSecret1);
    
    expect(verifyMacaroon(macaroon, rootSecret2)).toBe(false);
  });

  it("should successfully verify an attenuated macaroon", () => {
    const rootSecret = generateRootSecret();
    let macaroon = createMacaroon("key_123", rootSecret);
    
    macaroon = addCaveat(macaroon, "time < 2026-12-31");
    macaroon = addCaveat(macaroon, "ip = 192.168.1.1");
    
    expect(macaroon.caveats).toHaveLength(2);
    expect(verifyMacaroon(macaroon, rootSecret)).toBe(true);
  });

  it("should fail verification if caveats are tampered with", () => {
    const rootSecret = generateRootSecret();
    let macaroon = createMacaroon("key_123", rootSecret);
    
    macaroon = addCaveat(macaroon, "role = admin");
    
    // Tamper the caveat string
    macaroon.caveats[0] = "role = superadmin";
    
    expect(verifyMacaroon(macaroon, rootSecret)).toBe(false);
  });

  it("should fail verification if signature is tampered with", () => {
    const rootSecret = generateRootSecret();
    const macaroon = createMacaroon("key_123", rootSecret);
    
    macaroon.signature = "bad" + macaroon.signature.substring(3);
    
    expect(verifyMacaroon(macaroon, rootSecret)).toBe(false);
  });

  it("should serialize and deserialize correctly", () => {
    const rootSecret = generateRootSecret();
    let macaroon = createMacaroon("key_123", rootSecret);
    macaroon = addCaveat(macaroon, "some caveat");
    
    const token = serializeMacaroon(macaroon);
    const restored = deserializeMacaroon(token);
    
    expect(restored.id).toBe(macaroon.id);
    expect(restored.caveats).toEqual(macaroon.caveats);
    expect(restored.signature).toBe(macaroon.signature);
    expect(verifyMacaroon(restored, rootSecret)).toBe(true);
  });
});
