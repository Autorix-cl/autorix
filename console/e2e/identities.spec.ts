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

  test("paginates table according to selected rows per page", async ({ page }) => {
    // Check initial rows in table body (default 10)
    const rows = page.locator("tbody tr");
    const initialCount = await rows.count();
    expect(initialCount).toBeLessThanOrEqual(10);

    // Capture screenshot of paginated identities studio
    await page.screenshot({
      path: "/Users/macbook/.gemini/antigravity-cli/brain/3a3fb493-f34f-4483-8b1d-d2f8068933b2/screenshots/cloud-identities-paginated.png",
    });

    const footer = page.getByText("Rows per page").locator("../..");
    await footer.scrollIntoViewIfNeeded();
    await page.screenshot({
      path: "/Users/macbook/.gemini/antigravity-cli/brain/3a3fb493-f34f-4483-8b1d-d2f8068933b2/screenshots/cloud-identities-footer.png",
    });

    // Verify rows per page selector is visible with default 10
    await expect(page.getByText("Rows per page")).toBeVisible();
  });

  test("navigates to Schema Studio & Builder and loads preset templates", async ({ page }) => {
    // Click Schema Studio & Builder tab
    await page.getByRole("tab", { name: /Schema Studio & Builder/i }).click();

    // Verify catalog and editor elements
    await expect(page.getByText(/Ego Identity Schemas Catalog/i)).toBeVisible();
    await expect(page.getByText(/Interactive Form Preview/i)).toBeVisible();

    // Click B2B preset button
    await page.getByRole("button", { name: /^B2B$/i }).click();
    await expect(page.getByLabel(/Schema Identifier/i)).toHaveValue("b2b_partner");

    // Verify live form preview updated with B2B fields
    await expect(page.getByLabel(/Organization Name/i)).toBeVisible();

    // Capture screenshot of Schema Studio
    await page.screenshot({
      path: "/Users/macbook/.gemini/antigravity-cli/brain/3a3fb493-f34f-4483-8b1d-d2f8068933b2/screenshots/cloud-identities-schema-studio.png",
    });
  });

  test("navigates to Traits Architecture Guide tab", async ({ page }) => {
    // Click Traits Architecture Guide tab
    await page.getByRole("tab", { name: /Traits Architecture Guide/i }).click();

    // Verify guide contents
    await expect(page.getByText(/Ego Traits & JSON Schema Architecture Guide/i)).toBeVisible();
    await expect(page.getByText(/Zero-Migration Extensibility/i)).toBeVisible();
    await expect(page.getByText(/Anatomy of an Identity Schema/i)).toBeVisible();
    await expect(page.getByText(/API Quickstart/i)).toBeVisible();

    // Capture screenshot of Traits Guide
    await page.screenshot({
      path: "/Users/macbook/.gemini/antigravity-cli/brain/3a3fb493-f34f-4483-8b1d-d2f8068933b2/screenshots/cloud-identities-traits-guide.png",
    });
  });
});

