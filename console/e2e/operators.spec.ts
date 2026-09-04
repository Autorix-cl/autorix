import { test, expect } from "./fixtures";

test.describe("Argus Operator RBAC & Directory", () => {
  test("renders operators directory with master break-glass administrator", async ({ page }) => {
    await page.goto("/operators");
    await page.waitForLoadState("networkidle");

    await expect(page.getByRole("heading", { level: 1 })).toHaveText(/console operators|operators/i);
    await expect(page.getByText(/break-glass sovereignty policy|break-glass/i).first()).toBeVisible();
    await expect(page.locator("main").getByText("admin@autorix.local")).toBeVisible();
  });

  test("provisions a new operator with assigned role", async ({ page }) => {
    await page.goto("/operators");
    await page.waitForLoadState("networkidle");

    // Click Provision Operator button
    await page.getByRole("button", { name: /provision operator/i }).click();

    // Sheet should be visible
    await expect(page.getByRole("heading", { name: /provision new operator/i })).toBeVisible();

    // Fill form
    const timestamp = Date.now();
    await page.locator("#op-name").fill(`Auditor Test ${timestamp}`);
    await page.locator("#op-email").fill(`auditor-${timestamp}@autorix.io`);
    await page.locator("#op-password").fill("SuperSecret123!");

    // Role select (defaults to operator, let's select auditor)
    await page.locator("#op-role").click();
    await page.getByRole("option", { name: /auditor/i }).click();

    // Submit
    await page.getByRole("button", { name: /create operator/i }).click();

    // Expect success toast or newly created operator to be visible
    await expect(page.locator("main").getByText(`auditor-${timestamp}@autorix.io`)).toBeVisible({ timeout: 10000 });
  });

  test("redirects unauthenticated session to login page", async ({ page, context }) => {
    // Clear all cookies to simulate expired/lost session
    await context.clearCookies();

    // Attempting to navigate to operators page
    await page.goto("/operators");

    // Must be redirected to /login with ?from=/operators
    await page.waitForURL(/\/login\?from=%2Foperators/);
    await expect(page.getByRole("heading", { name: /autorix console/i })).toBeVisible();
  });
});
