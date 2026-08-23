import crypto from "crypto";

// Generates a secure random 32-byte secret (Base64 URL safe)
export function generateRootSecret(): string {
  return crypto.randomBytes(32).toString("base64url");
}

// Generate a Macaroon
export interface Macaroon {
  id: string; // The root key ID
  location: string;
  caveats: string[];
  signature: string; // hex
}

// Creates a new Macaroon from a Root Key
export function createMacaroon(rootKeyId: string, rootSecret: string, location: string = "autorix.console"): Macaroon {
  const signature = crypto.createHmac("sha256", rootSecret).update(rootKeyId).digest("hex");
  return {
    id: rootKeyId,
    location,
    caveats: [],
    signature,
  };
}

// Attenuate a Macaroon by adding a caveat
export function addCaveat(macaroon: Macaroon, caveat: string): Macaroon {
  // A caveat updates the signature using the previous signature as the key
  const newSignature = crypto
    .createHmac("sha256", Buffer.from(macaroon.signature, "hex"))
    .update(caveat)
    .digest("hex");

  return {
    ...macaroon,
    caveats: [...macaroon.caveats, caveat],
    signature: newSignature,
  };
}

// Verify a Macaroon given the root secret
export function verifyMacaroon(macaroon: Macaroon, rootSecret: string): boolean {
  // 1. Recompute the root signature
  let currentSig = crypto.createHmac("sha256", rootSecret).update(macaroon.id).digest("hex");

  // 2. Reapply all caveats sequentially
  for (const caveat of macaroon.caveats) {
    currentSig = crypto
      .createHmac("sha256", Buffer.from(currentSig, "hex"))
      .update(caveat)
      .digest("hex");
  }

  // 3. Compare final signature safely (prevent timing attacks)
  try {
    return crypto.timingSafeEqual(
      Buffer.from(currentSig, "hex"),
      Buffer.from(macaroon.signature, "hex")
    );
  } catch {
    return false;
  }
}

// Serialize to Base64URL string for HTTP headers
export function serializeMacaroon(macaroon: Macaroon): string {
  return Buffer.from(JSON.stringify(macaroon)).toString("base64url");
}

// Deserialize from Base64URL string
export function deserializeMacaroon(token: string): Macaroon {
  const json = Buffer.from(token, "base64url").toString("utf-8");
  return JSON.parse(json) as Macaroon;
}
