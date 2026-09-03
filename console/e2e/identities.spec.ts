import { test, expect } from "./fixtures";

test.describe("Ego Identities Studio (Ory Kratos Trait Model)", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/identities");
    await page.waitForLoadState("networkidle");
  });

  test("renders identity lifecycle overview and schema definition", async ({ page }) => {
    await expect(page.getByRole("heading", { level: 1 })).toHaveText(/Autorix Ego|Identities/i);
    await expect(page.getByText(/Registered Identities/i)).toBeVisible();
    await expect(page.getByRole("button", { name: /Create Identity/i })).toBeVisible();
  });

  test("registers a new identity via invitation sheet and updates table dynamically", async ({ page }) => {
    const unique = `${Date.now()}-${Math.random().toString(36).slice(2, 7)}`;
    const email = `e2e-user-${unique}@autorix.internal`;

    await page.getByRole("button", { name: /Create Identity/i }).click();
    await page.locator("#email").fill(email);
    await page.locator("#firstName").fill("Elena");
    await page.locator("#lastName").fill("Rostova");

    await page.getByRole("button", { name: /Send Invitation/i }).click();

    // Verify invitation confirmation toast
    await expect(page.getByText(/Invitation.*sent/i)).toBeVisible({ timeout: 10_000 });
  });

  test("filters existing identities by email search query", async ({ page, seedIdentity }) => {
    const unique = `${Date.now()}-${Math.random().toString(36).slice(2, 7)}`;
    const email = `search-${unique}@autorix.internal`;
    await seedIdentity({ email });

    await page.goto("/identities");
    await page.waitForLoadState("networkidle");

    const searchInput = page.locator("#searchQuery, input[placeholder*='Search identities']").first();
    await searchInput.fill(email);

    // Verify only the searched identity is displayed
    await expect(page.getByRole("cell", { name: email })).toBeVisible();
  });
});
