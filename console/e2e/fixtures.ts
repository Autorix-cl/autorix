import { test as base, expect } from "@playwright/test";

/**
 * Shared E2E fixtures (P1-S5-T5). seedIdentity creates a real identity
 * through the console's own /api/identities BFF route — exercising the
 * exact same path a user's browser takes, not a shortcut straight to ego —
 * so specs that need existing data don't depend on execution order or
 * leftover state from a previous run.
 */
interface Fixtures {
  seedIdentity: (overrides?: {
    email?: string;
    password?: string;
    firstName?: string;
    lastName?: string;
  }) => Promise<{ email: string; firstName: string; lastName: string }>;
}

export const test = base.extend<Fixtures>({
  seedIdentity: async ({ request }, use) => {
    await use(async (overrides = {}) => {
      const unique = `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
      const email = overrides.email ?? `e2e-${unique}@autorix.io`;
      const firstName = overrides.firstName ?? "E2E";
      const lastName = overrides.lastName ?? `Test-${unique}`;

      const res = await request.post("/api/identities", {
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
});

export { expect };
