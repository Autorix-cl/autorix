import { test, expect } from "./fixtures";

test.describe("Janus OAuth2 / OIDC Studio", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/oauth2");
    await page.waitForLoadState("networkidle");
  });

  test("renders client management and JWKS cryptographic key set viewer", async ({ page }) => {
    await expect(page.getByRole("heading", { level: 1 })).toHaveText(/Autorix Janus|OAuth2/i);
    await expect(page.getByText(/Register OAuth2 Application|Registered OAuth2 Clients/i).first()).toBeVisible();
    await expect(page.getByText(/Active JWKS Public Key|JWKS/i).first()).toBeVisible();
  });

  test("registers a new OAuth2 client application and updates table", async ({ page }) => {
    const unique = `${Date.now()}-${Math.random().toString(36).slice(2, 7)}`;
    const clientName = `E2E Service ${unique}`;
    const clientId = `app-${unique}`;

    await page.locator("#clientId").fill(clientId);
    await page.locator("#clientName").fill(clientName);
    await page.locator("#clientSecret").fill("SuperSecretSecret123!");
    await page.getByRole("button", { name: /Register Application/i }).click();

    // Verify newly created client appears in table
    await expect(page.getByRole("cell", { name: clientId })).toBeVisible({ timeout: 10_000 });
  });
});
