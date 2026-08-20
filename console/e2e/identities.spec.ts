import { test, expect } from "./fixtures";

test.describe("Ego Identities Studio", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/identities");
    await page.waitForLoadState("networkidle");
  });

  test("renders identity lifecycle overview and schema definition", async ({ page }) => {
    await expect(page.getByRole("heading", { level: 1 })).toHaveText(/Autorix Ego|Identity/i);
    await expect(page.getByText(/Dynamic Profile Schema|JSON Schema/i).first()).toBeVisible();
  });

  test("registers a new identity via self-service form and updates table dynamically", async ({ page }) => {
    const unique = `${Date.now()}-${Math.random().toString(36).slice(2, 7)}`;
    const email = `test-user-${unique}@autorix.io`;

    await page.locator("#email").fill(email);
    await page.locator("#firstName").fill("E2E");
    await page.locator("#lastName").fill(`User-${unique}`);
    await page.locator("#password").fill("SecurePassword123!");
    await page.getByRole("button", { name: /Provision User & Hash Password|Provision User/i }).click();

    // Verify identity appears in table
    await expect(page.getByRole("cell", { name: email })).toBeVisible({ timeout: 10_000 });
  });

  test("filters existing identities by email search query", async ({ page, seedIdentity }) => {
    const identity = await seedIdentity();

    await page.goto("/identities");
    await page.waitForLoadState("networkidle");

    const searchInput = page.getByPlaceholder(/Filter identities/i);
    await searchInput.fill(identity.email);

    await expect(page.getByText(identity.email)).toBeVisible();
  });
});
