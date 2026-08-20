import { test, expect } from "@playwright/test";

test.describe("Authentication Flows", () => {
  test.use({ storageState: { cookies: [], origins: [] } });

  test("unauthenticated visitor to protected route is redirected to /login with return query parameter", async ({ page }) => {
    await page.goto("/identities");
    await expect(page).toHaveURL(/\/login\?from=%2Fidentities/);
    await expect(page.getByRole("heading", { level: 1 })).toHaveText(/autorix console/i);
    await expect(page.getByRole("button", { name: /break-glass operator/i })).toBeVisible();
    await expect(page.getByRole("button", { name: /enterprise sso/i })).toBeVisible();
  });

  test("operator login succeeds with valid credentials and redirects to dashboard", async ({ page }) => {
    await page.goto("/login");
    
    const emailInput = page.locator('input[type="email"]');
    const passwordInput = page.locator('input[type="password"]');
    const submitBtn = page.getByRole("button", { name: /authenticate session/i });

    await emailInput.fill("admin@autorix.local");
    await passwordInput.fill("SecretMasterKey#2026");
    await submitBtn.click();

    await page.waitForURL((url) => !url.pathname.startsWith("/login"), { timeout: 15_000 });
    await expect(page).toHaveURL(/localhost:3000\/?$/);
    await expect(page.getByRole("heading", { level: 1 })).toHaveText(/telemetry|overview|autorix/i);
  });

  test("operator login fails with invalid credentials and displays error message", async ({ page }) => {
    await page.goto("/login");

    const emailInput = page.locator('input[type="email"]');
    const passwordInput = page.locator('input[type="password"]');
    const submitBtn = page.getByRole("button", { name: /authenticate session/i });

    await emailInput.fill("admin@autorix.local");
    await passwordInput.fill("CompletelyWrongPassword123!");
    await submitBtn.click();

    // Verify error notification or inline alert
    await expect(page.locator(".bg-red-500\\/10, [role='status']").first()).toBeVisible({ timeout: 5000 });
    await expect(page).toHaveURL(/\/login/);
  });

  test("password visibility toggle reveals and hides plaintext password", async ({ page }) => {
    await page.goto("/login");
    const passwordInput = page.locator('input[type="password"]');
    await passwordInput.fill("Secret12345");

    const toggleBtn = page.locator('button:has(svg.lucide-eye, svg.lucide-eye-off)');
    await expect(toggleBtn).toBeVisible();
    await toggleBtn.click();
    await expect(page.locator('input[type="text"]').filter({ hasNotText: "" })).toBeVisible();
  });

  test("enterprise SSO tab provides identity provider routing scaffold", async ({ page }) => {
    await page.goto("/login");
    const ssoTab = page.getByRole("button", { name: /enterprise sso/i });
    await ssoTab.click();

    await expect(page.getByText(/corporate single sign-on/i)).toBeVisible();
    await expect(page.getByRole("button", { name: /continue with enterprise sso/i })).toBeVisible();
  });
});
