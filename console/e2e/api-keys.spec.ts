import { test, expect } from "./fixtures";

test.describe("Vulcan API Keys & Macaroons Studio", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/api-keys");
    await page.waitForLoadState("networkidle");
  });

  test("renders API key generation and macaroon attenuation studio", async ({ page }) => {
    await expect(page.getByRole("heading", { level: 1 })).toHaveText(/Autorix Vulcan|API Keys/i);
    await expect(page.locator("#keyName")).toBeVisible();
    await expect(page.locator("#ownerId")).toBeVisible();
  });

  test("creates a new API key and reveals the secret token once", async ({ page }) => {
    const unique = `${Date.now()}-${Math.random().toString(36).slice(2, 7)}`;
    const keyName = `e2e-key-${unique}`;

    await page.locator("#keyName").fill(keyName);
    await page.locator("#ownerId").fill(`owner-${unique}`);
    await page.getByRole("button", { name: /Generate Scannable API Key|Generate API Key/i }).click();

    // Verify secret token reveal box appears
    await expect(page.getByText(/API Key Generated|PLAINTEXT VULCAN API KEY/i).first()).toBeVisible({ timeout: 10_000 });
  });
});
