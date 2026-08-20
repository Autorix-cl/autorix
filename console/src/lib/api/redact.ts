/**
 * Secret redaction discipline for console logging and errors (P3-S5-T6).
 * Strips credentials, tokens, keys and hashes so secrets never leak into logs or telemetry.
 */

const SECRET_PATTERNS = [
  // Argus tokens
  /\b(abt_[a-f0-9]{32,})\b/gi,
  /\b(ast_[a-f0-9]{32,})\b/gi,
  /\b(aet_[a-f0-9]{32,})\b/gi,
  /\b(apt_[a-f0-9]{32,})\b/gi,
  /\b(asa_[a-f0-9]{32,})\b/gi,
  // Vulcan keys
  /\b(av_live_[a-zA-Z0-9_-]{24,})\b/gi,
  /\b(av_test_[a-zA-Z0-9_-]{24,})\b/gi,
  // Argon2 hashes
  /\$argon2id\$v=\d+\$m=\d+,t=\d+,p=\d+\$[a-zA-Z0-9+/=]+\$[a-zA-Z0-9+/=]+/gi,
  // Bearer tokens
  /Bearer\s+[a-zA-Z0-9_.-]{20,}/gi,
];

export function redactSecrets(input: string): string {
  let output = input;
  for (const pattern of SECRET_PATTERNS) {
    output = output.replace(pattern, "[REDACTED_SECRET]");
  }
  return output;
}

export function redactObject<T>(obj: T): T {
  if (!obj || typeof obj !== "object") {
    if (typeof obj === "string") {
      return redactSecrets(obj) as unknown as T;
    }
    return obj;
  }

  if (Array.isArray(obj)) {
    return obj.map((item) => redactObject(item)) as unknown as T;
  }

  const result: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(obj as Record<string, unknown>)) {
    const lowerKey = key.toLowerCase();
    if (
      lowerKey.includes("password") ||
      lowerKey.includes("secret") ||
      lowerKey.includes("token") ||
      lowerKey.includes("key_hash") ||
      lowerKey.includes("private_key")
    ) {
      result[key] = "[REDACTED]";
    } else {
      result[key] = redactObject(value);
    }
  }
  return result as T;
}
