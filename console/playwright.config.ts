import { defineConfig, devices } from "@playwright/test";

/**
 * E2E scaffold against the composed stack (P1-S5-T5). Assumes the full
 * `docker compose` stack (7 engines + console) is already up and healthy —
 * these specs exercise real navigation and real data, not a mocked
 * backend, so they belong after `scripts/ci/smoke_test.sh` in CI, and
 * against a locally running `docker compose up` for local development.
 */
export default defineConfig({
  testDir: "./e2e",
  globalSetup: "./e2e/global-setup.ts",
  fullyParallel: false,
  workers: 1,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? [["github"], ["html", { open: "never" }]] : "list",
  use: {
    baseURL: process.env.PLAYWRIGHT_BASE_URL ?? "http://localhost:3000",
    storageState: "./e2e/.auth/operator.json",
    trace: "on-first-retry",
  },
  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
    },
  ],
});
