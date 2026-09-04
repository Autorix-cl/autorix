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

  test("switches between studio tabs and captures visual states", async ({ page }) => {
    await page.setViewportSize({ width: 1440, height: 1100 });
    // Initial tab: API Keys Vault
    await expect(page.getByRole("tab", { name: /API Keys Vault/i })).toBeVisible();
    await expect(page.getByText(/API Key Registry & Zero-Downtime Rotation/i)).toBeVisible();

    // Capture screenshot of API Keys tab
    await page.screenshot({
      path: "/Users/macbook/.gemini/antigravity-cli/brain/3a3fb493-f34f-4483-8b1d-d2f8068933b2/screenshots/cloud-vulcan-keys.png",
    });

    // Switch to Macaroon Inspector tab
    await page.getByRole("tab", { name: /Macaroon Inspector/i }).click();
    await expect(page.getByText(/Capability Token Inspector & Live Verification/i)).toBeVisible();

    // Capture screenshot of Macaroon Inspector tab
    await page.screenshot({
      path: "/Users/macbook/.gemini/antigravity-cli/brain/3a3fb493-f34f-4483-8b1d-d2f8068933b2/screenshots/cloud-vulcan-inspector.png",
      fullPage: true,
    });

    // Load sample and test decode
    await page.getByRole("button", { name: /Load Sample Macaroon/i }).click();
    await page.getByRole("button", { name: "Decode" }).click();
    await expect(page.getByText("Decoded Structure")).toBeVisible();

    // Capture screenshot with decoded structure and verification playground
    await page.screenshot({
      path: "/Users/macbook/.gemini/antigravity-cli/brain/3a3fb493-f34f-4483-8b1d-d2f8068933b2/screenshots/cloud-vulcan-inspector-decoded.png",
      fullPage: true,
    });

    // Switch to Attenuation Studio tab
    await page.getByRole("tab", { name: /Attenuation Studio/i }).click();
    await expect(page.getByText(/Cryptographic Attenuation Studio/i)).toBeVisible();

    // Capture screenshot of Attenuation Studio tab
    await page.screenshot({
      path: "/Users/macbook/.gemini/antigravity-cli/brain/3a3fb493-f34f-4483-8b1d-d2f8068933b2/screenshots/cloud-vulcan-attenuation.png",
    });
  });
});
