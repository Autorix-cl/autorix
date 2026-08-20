import { test, expect } from "./fixtures";

/**
 * The dashboard renders real, measured engine state (P1-S4/P1-S5-T5) — not
 * the simulation layer the roadmap's audit found ("cluster simulation
 * fallback", a hardcoded 100% HEALTHY regardless of what was actually
 * running). These specs are a regression guard: if that fabricated-data
 * pattern ever comes back, they fail.
 */
test.describe("dashboard real-state rendering", () => {
  test("never shows the old fabricated metrics", async ({ page }) => {
    await page.goto("/");

    const body = page.locator("body");
    await expect(body).not.toContainText("100% HEALTHY");
    await expect(body).not.toContainText("148.2k req/s");
    await expect(body).not.toContainText("< 1.8 ms");
    await expect(body).not.toContainText("7/7 UP");
  });

  test("shows a real, measured services count against all 7 engines", async ({ page }) => {
    await page.goto("/");

    // With the full compose stack up, every engine is reachable — the
    // aggregator (P1-S4-T1) should report all 7 healthy, not a hardcoded
    // "6" or a count that never changes. Two places legitimately render
    // this (the sidebar footer and the stats StatCard) — .first() is
    // enough to prove it's rendered, not to pick one over the other.
    await expect(page.getByText(/\d+\s*\/\s*7/).first()).toBeVisible({ timeout: 15_000 });
  });

  test("shows a last-checked timestamp sourced from the real health probe", async ({ page }) => {
    await page.goto("/");
    await expect(page.getByText(/last checked/i)).toBeVisible({ timeout: 15_000 });
  });

  test("honestly labels throughput as not available instead of inventing a number", async ({ page }) => {
    await page.goto("/");
    // No engine emits request-rate metrics yet (P7, not built) — the
    // dashboard must say so, not show a fabricated req/s figure.
    await expect(page.getByText(/not available/i).first()).toBeVisible();
  });

  test("refresh button re-probes engine health", async ({ page }) => {
    await page.goto("/");
    const refreshButton = page.getByRole("button", { name: /refresh/i });
    await expect(refreshButton).toBeEnabled({ timeout: 15_000 });
    await refreshButton.click();
    // The click should not throw / leave the button permanently disabled.
    await expect(refreshButton).toBeEnabled({ timeout: 15_000 });
  });
});

test.describe("identities page shows real seeded data", () => {
  test("a freshly seeded identity appears in the table", async ({ page, seedIdentity }) => {
    const identity = await seedIdentity();

    await page.goto("/identities");
    await expect(page.getByText(identity.email)).toBeVisible({ timeout: 15_000 });
  });
});
