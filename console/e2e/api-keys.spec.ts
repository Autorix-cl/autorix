import { test, expect } from "./fixtures";

test.describe("Vulcan API Keys & Macaroons Studio", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/vulcan");
    await page.waitForLoadState("networkidle");
  });

  test("renders API key generation and macaroon attenuation studio", async ({ page }) => {
    await expect(page.getByRole("heading", { level: 1 })).toHaveText(/Vulcan \(API Keys & Macaroons\)/i);
    await expect(page.getByRole("button", { name: /Create API Key/i })).toBeVisible();
    await expect(page.getByText(/Attenuation Studio/i).first()).toBeVisible();
  });

  test("creates a new API key and reveals secret token modal", async ({ page }) => {
    const unique = `${Date.now()}-${Math.random().toString(36).slice(2, 7)}`;
    const keyName = `e2e-key-${unique}`;

    await page.getByRole("button", { name: /Create API Key/i }).click();
    await page.locator("#name").fill(keyName);
    await page.getByLabel(/Read Users/i).check();
    await page.getByRole("button", { name: /Generate Key/i }).click();

    // Verify secret modal appears
    await expect(page.getByText(/Save your secret key/i)).toBeVisible({ timeout: 10_000 });
  });
});
