import test from "node:test";
import assert from "node:assert/strict";
import {
  IDENTITY_SIGNATURE_HEADER,
  extractUserFromHeaders,
  signIdentityHeaders,
  verifyIdentityHeaders,
} from "../dist/middleware.js";

test("extractUserFromHeaders verifies signed Aegis identity headers", () => {
  const secret = "ts-proxy-secret";
  const sig = signIdentityHeaders("usr_42", "alice@autorix.io", "admin, viewer", "read write", secret);
  assert.ok(sig.startsWith("v1="));

  const headers = {
    "x-user-id": "usr_42",
    "x-user-email": "alice@autorix.io",
    "x-user-roles": "admin, viewer",
    "x-user-scopes": "read write",
    [IDENTITY_SIGNATURE_HEADER]: sig,
  };

  assert.equal(verifyIdentityHeaders(headers, secret), true);
  const user = extractUserFromHeaders(headers, { signingSecret: secret });
  assert.deepEqual(user, {
    id: "usr_42",
    email: "alice@autorix.io",
    roles: ["admin", "viewer"],
  });

  // Tampered header rejected
  assert.equal(
    extractUserFromHeaders({ ...headers, "x-user-id": "usr_admin" }, { signingSecret: secret }),
    null,
  );

  // Unsigned header rejected when secret is configured
  assert.equal(
    extractUserFromHeaders({ "x-user-id": "usr_spoofed" }, { signingSecret: secret }),
    null,
  );

  // Fail-closed when requireSignature is true without secret
  assert.equal(
    extractUserFromHeaders({ "x-user-id": "usr_spoofed" }, { signingSecret: "", requireSignature: true }),
    null,
  );
});
