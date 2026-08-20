import { test as base, expect, type APIRequestContext } from "@playwright/test";
import * as fs from "fs";
import * as path from "path";

/**
 * Reads the CSRF token from the operator.json storageState file or performs a GET
 * to acquire the CSRF cookie for mutating BFF requests.
 */
async function getCsrfToken(request: APIRequestContext): Promise<string> {
  try {
    const authFile = path.join(__dirname, ".auth", "operator.json");
    if (fs.existsSync(authFile)) {
      const data = JSON.parse(fs.readFileSync(authFile, "utf-8"));
      const csrfCookie = data.cookies?.find((c: { name: string }) => c.name === "autorix_csrf");
      if (csrfCookie?.value) {
        return csrfCookie.value;
      }
    }
  } catch {}

  const res = await request.get("/api/health");
  const cookies = res.headers()["set-cookie"] || "";
  const match = cookies.match(/autorix_csrf=([a-f0-9]+)/);
  return match ? match[1] : "e2e-csrf-fallback";
}

interface Fixtures {
  seedIdentity: (overrides?: {
    email?: string;
    password?: string;
    firstName?: string;
    lastName?: string;
  }) => Promise<{ email: string; firstName: string; lastName: string }>;

  seedOAuth2Client: (overrides?: {
    client_name?: string;
    client_type?: string;
    scope?: string;
  }) => Promise<{ client_id: string; client_name: string }>;

  seedPolicy: (overrides?: {
    name?: string;
    expression?: string;
    priority?: number;
  }) => Promise<{ id: string; name: string }>;

  seedProxyRule: (overrides?: {
    match_path?: string;
    match_methods?: string[];
    upstream_url?: string;
  }) => Promise<{ id: string; match_path: string }>;

  seedApiKey: (overrides?: {
    name?: string;
    owner_id?: string;
  }) => Promise<{ id: string; name: string }>;
}

export const test = base.extend<Fixtures>({
  seedIdentity: async ({ request }, use) => {
    await use(async (overrides = {}) => {
      const csrf = await getCsrfToken(request);
      const unique = `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
      const email = overrides.email ?? `e2e-${unique}@autorix.io`;
      const firstName = overrides.firstName ?? "E2E";
      const lastName = overrides.lastName ?? `Test-${unique}`;

      const res = await request.post("/api/identities", {
        headers: {
          "X-CSRF-Token": csrf,
        },
        data: {
          email,
          password: overrides.password ?? "e2e-test-password-1",
          firstName,
          lastName,
        },
      });
      expect(
        res.ok(),
        `seedIdentity: POST /api/identities failed (status ${res.status()}): ${await res.text()}`,
      ).toBeTruthy();

      return { email, firstName, lastName };
    });
  },

  seedOAuth2Client: async ({ request }, use) => {
    await use(async (overrides = {}) => {
      const csrf = await getCsrfToken(request);
      const unique = `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
      const clientName = overrides.client_name ?? `E2E Client ${unique}`;

      const res = await request.post("/api/oauth2/clients", {
        headers: {
          "X-CSRF-Token": csrf,
        },
        data: {
          client_name: clientName,
          client_type: overrides.client_type ?? "confidential",
          scope: overrides.scope ?? "openid profile email",
        },
      });
      expect(
        res.ok(),
        `seedOAuth2Client: POST /api/oauth2/clients failed (status ${res.status()}): ${await res.text()}`,
      ).toBeTruthy();

      const data = await res.json();
      return { client_id: data.client_id || unique, client_name: clientName };
    });
  },

  seedPolicy: async ({ request }, use) => {
    await use(async (overrides = {}) => {
      const csrf = await getCsrfToken(request);
      const unique = `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
      const name = overrides.name ?? `E2E Policy ${unique}`;

      const res = await request.post("/api/policies", {
        headers: {
          "X-CSRF-Token": csrf,
        },
        data: {
          name,
          expression: overrides.expression ?? 'request.auth.claims.department == "engineering"',
          priority: overrides.priority ?? 1,
          description: "E2E automated test policy",
        },
      });
      expect(
        res.ok(),
        `seedPolicy: POST /api/policies failed (status ${res.status()}): ${await res.text()}`,
      ).toBeTruthy();

      const data = await res.json();
      return { id: data.id || unique, name };
    });
  },

  seedProxyRule: async ({ request }, use) => {
    await use(async (overrides = {}) => {
      const csrf = await getCsrfToken(request);
      const unique = `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
      const matchPath = overrides.match_path ?? `/api/e2e/${unique}/*`;

      const res = await request.post("/api/proxy-rules", {
        headers: {
          "X-CSRF-Token": csrf,
        },
        data: {
          id: unique,
          match: {
            url: matchPath,
            methods: overrides.match_methods ?? ["GET", "POST"],
          },
          authenticators: [{ handler: "jwt" }],
          authorizer: { handler: "allow" },
          mutators: [{ handler: "header" }],
          upstream: {
            url: overrides.upstream_url ?? "http://localhost:8080",
          },
        },
      });
      expect(
        res.ok(),
        `seedProxyRule: POST /api/proxy-rules failed (status ${res.status()}): ${await res.text()}`,
      ).toBeTruthy();

      const data = await res.json();
      return { id: data.id || unique, match_path: matchPath };
    });
  },

  seedApiKey: async ({ request }, use) => {
    await use(async (overrides = {}) => {
      const csrf = await getCsrfToken(request);
      const unique = `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
      const name = overrides.name ?? `e2e_key_${unique}`;

      const res = await request.post("/api/keys", {
        headers: {
          "X-CSRF-Token": csrf,
        },
        data: {
          name,
          owner_id: overrides.owner_id ?? `user_${unique}`,
          scopes: ["read", "write"],
        },
      });
      expect(
        res.ok(),
        `seedApiKey: POST /api/keys failed (status ${res.status()}): ${await res.text()}`,
      ).toBeTruthy();

      const data = await res.json();
      return { id: data.id || unique, name };
    });
  },
});

export { expect };
