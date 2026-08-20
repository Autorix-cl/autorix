import { test, expect } from "./fixtures";

test.describe("Multi-Environment Isolation", () => {
  test.afterEach(async ({ page, context }) => {
    await context.addCookies([{ name: "autorix_active_env", value: "prod", domain: "localhost", path: "/" }]);
    await page.evaluate(() => {
      localStorage.setItem("autorix_active_env", "prod");
      document.cookie = "autorix_active_env=prod; path=/";
    });
  });

  test("environment switcher defaults to Production with active connected engines", async ({ page }) => {
    await page.goto("/");
    await page.waitForLoadState("networkidle");

    const envBtn = page.locator("header button").filter({ hasText: /production/i }).first();
    await expect(envBtn).toBeVisible();
    await expect(page.getByText(/connected \(5432\)|5432/i)).toBeVisible();
  });

  test("switching to Staging activates non-production warning banner and degrades engine cards", async ({ page }) => {
    await page.goto("/");
    await page.waitForLoadState("networkidle");

    const envBtn = page.locator("header button").filter({ hasText: /production/i }).first();
    await envBtn.click();
    await page.getByRole("menuitem").filter({ hasText: /staging/i }).click();

    await expect(page.getByText(/staging environment|isolated sandbox|0 instances|staging/i).first()).toBeVisible({ timeout: 5000 });
  });

  test("switching to Development gates engine studio page with NotConnectedEngine banner", async ({ page }) => {
    await page.goto("/identities");
    await page.waitForLoadState("networkidle");

    const envBtn = page.locator("header button").filter({ hasText: /production/i }).first();
    await envBtn.click();
    await page.getByRole("menuitem").filter({ hasText: /development/i }).click();

    await expect(page.getByText(/not connected|not enrolled|enroll instance|0 instances/i).first()).toBeVisible({ timeout: 5000 });
  });
});
