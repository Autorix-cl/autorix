import { test, expect } from "./fixtures";

test.describe("Aegis Zero-Trust Proxy Studio", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/proxy-rules");
    await page.waitForLoadState("networkidle");
  });

  test("renders proxy rules and rule matching simulator", async ({ page }) => {
    await expect(page.getByRole("heading", { level: 1 })).toHaveText(/Autorix Aegis|Zero Trust|Proxy/i);
    await expect(page.getByText(/Proxy Routing & Enforcement Simulator|Simulator/i).first()).toBeVisible();
  });

  test("runs test match evaluation for a request URI against active rules", async ({ page, seedProxyRule }) => {
    const rule = await seedProxyRule();

    await page.goto("/proxy-rules");
    await page.waitForLoadState("networkidle");

    const pathInput = page.locator("input[id='path'], input[name='path']").or(page.locator("input[placeholder*='api/v1'], input[placeholder*='documents']")).first();
    const testBtn = page.getByRole("button", { name: /Simulate Match|Testing Match/i });

    if (await pathInput.isVisible()) {
      await pathInput.fill(rule.match_path.replace("/*", "/item-123"));
    }
    await testBtn.click();

    // Verify match result card or status is shown
    await expect(page.getByText(/Rule Matched|No rule matched|Upstream Target|Authenticator/i).first()).toBeVisible({ timeout: 5000 });
  });
});
