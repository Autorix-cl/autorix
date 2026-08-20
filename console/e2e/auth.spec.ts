import { test, expect } from "./fixtures";

test.describe("Authentication & Session Security (Argus Gateway)", () => {
  test("unauthenticated visitor to protected route is redirected to /login with return query parameter", async ({ page }) => {
    await page.context().clearCookies();
    await page.goto("/identities");

    await page.waitForURL((url) => url.pathname === "/login");
    const url = new URL(page.url());
    expect(url.searchParams.get("from")).toBe("/identities");
    await expect(page.getByRole("button", { name: /authenticate session/i })).toBeVisible();
    await expect(page.getByRole("button", { name: /enterprise sso/i })).toBeVisible();
  });

  test("deep-linking: operator login preserves and redirects to requested protected page", async ({ page }) => {
    await page.context().clearCookies();
    await page.goto("/login?from=%2Fpolicies");
    
    const emailInput = page.locator('input[type="email"]');
    const passwordInput = page.locator('input[type="password"]');
    const submitBtn = page.getByRole("button", { name: /authenticate session/i });

    await emailInput.fill("admin@autorix.local");
    await passwordInput.fill("SecretMasterKey#2026");
    await submitBtn.click();

    await page.waitForURL((url) => url.pathname === "/policies", { timeout: 15_000 });
    await expect(page).toHaveURL(/\/policies$/);
    await expect(page.getByRole("heading", { level: 1 })).toHaveText(/themis|cel policy/i);
  });

  test("open-redirect defense: prevents external redirect when from parameter is absolute URL", async ({ page }) => {
    await page.context().clearCookies();
    await page.goto("/login?from=https%3A%2F%2Fmalicious-site.com");

    const emailInput = page.locator('input[type="email"]');
    const passwordInput = page.locator('input[type="password"]');
    const submitBtn = page.getByRole("button", { name: /authenticate session/i });

    await emailInput.fill("admin@autorix.local");
    await passwordInput.fill("SecretMasterKey#2026");
    await submitBtn.click();

    // Must safely land on dashboard, NOT malicious site
    await page.waitForURL((url) => !url.pathname.startsWith("/login"), { timeout: 15_000 });
    expect(page.url()).not.toContain("malicious-site.com");
    await expect(page).toHaveURL(/localhost:3000\/?$/);
  });

  test("operator login fails with invalid credentials and displays error message", async ({ page }) => {
    await page.context().clearCookies();
    await page.goto("/login");

    const emailInput = page.locator('input[type="email"]');
    const passwordInput = page.locator('input[type="password"]');
    const submitBtn = page.getByRole("button", { name: /authenticate session/i });

    await emailInput.fill("admin@autorix.local");
    await passwordInput.fill("WrongPassword999!");
    await submitBtn.click();

    const alert = page.locator("[role='alert']").or(page.getByText(/invalid email or password|authentication failed/i));
    await expect(alert.first()).toBeVisible({ timeout: 5000 });
  });

  test("password visibility toggle reveals and hides plaintext password", async ({ page }) => {
    await page.context().clearCookies();
    await page.goto("/login");

    const passwordInput = page.locator('input[type="password"], input[type="text"][placeholder*="••"]').first();
    await passwordInput.fill("MySecretPassword123");
    await expect(passwordInput).toHaveAttribute("type", "password");

    const toggleBtn = page.locator("button:has(svg.lucide-eye, svg.lucide-eye-off)").first();
    await toggleBtn.click();
    await expect(passwordInput).toHaveAttribute("type", "text");

    await toggleBtn.click();
    await expect(passwordInput).toHaveAttribute("type", "password");
  });

  test("enterprise SSO tab provides identity provider routing scaffold", async ({ page }) => {
    await page.context().clearCookies();
    await page.goto("/login");

    const ssoTab = page.getByRole("button", { name: /enterprise sso/i });
    await ssoTab.click();

    await expect(page.getByText(/corporate single sign-on/i).first()).toBeVisible();
    await expect(page.getByRole("button", { name: /continue with enterprise sso/i })).toBeVisible();
  });
});
