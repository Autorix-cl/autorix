import { test, expect } from "./fixtures";

test.describe("Janus OAuth2 / OIDC Studio (Ory Hydra Protocol)", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/oauth2");
    await page.waitForLoadState("networkidle");
  });

  test("renders client management and JWKS cryptographic key set viewer", async ({ page }) => {
    await expect(page.getByRole("heading", { level: 1 })).toHaveText(/Autorix Janus|OAuth2/i);
    await expect(page.getByText(/Active JWKS Public Key/i).first()).toBeVisible();
    await expect(page.getByText(/Registered OAuth2 Clients/i).first()).toBeVisible();
  });

  test("registers a new OAuth2 client application and updates table", async ({ page }) => {
    const unique = `${Date.now()}-${Math.random().toString(36).slice(2, 7)}`;
    const clientId = `client-${unique}`;
    const clientName = `Production Microservice ${unique}`;

    await page.locator("#clientId").fill(clientId);
    await page.locator("#clientName").fill(clientName);
    await page.locator("#clientSecret").fill("SuperSecretClientSecret123!");
    await page.locator("#scopes").fill("openid profile email offline_access");

    await page.getByRole("button", { name: /Register Application/i }).click();

    // Verify client appears in table
    await expect(page.getByRole("cell", { name: clientId })).toBeVisible({ timeout: 10_000 });
  });

  test("registers a public SPA client without client secret (PKCE enforce)", async ({ page }) => {
    const unique = `${Date.now()}-${Math.random().toString(36).slice(2, 7)}`;
    const clientId = `spa-${unique}`;
    const clientName = `Frontend Single Page App ${unique}`;

    await page.locator("#clientId").fill(clientId);
    await page.locator("#clientName").fill(clientName);

    // Switch client type to public
    await page.locator("button#clientType").click();
    await page.getByRole("option", { name: /Public \(SPA/i }).click();

    // Client secret input must disappear for public clients
    await expect(page.locator("#clientSecret")).not.toBeVisible();

    await page.getByRole("button", { name: /Register Application/i }).click();

    // Verify public client in table
    await expect(page.getByRole("cell", { name: clientId })).toBeVisible({ timeout: 10_000 });
  });
});
