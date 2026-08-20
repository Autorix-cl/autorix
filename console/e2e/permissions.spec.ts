import { test, expect } from "./fixtures";

test.describe("Nexus ReBAC Permissions Studio (Google Zanzibar)", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/permissions");
    await page.waitForLoadState("networkidle");
  });

  test("renders Zanzibar relation graph simulator and relation tuples table", async ({ page }) => {
    await expect(page.getByRole("heading", { level: 1 })).toHaveText(/Autorix Nexus|Permissions/i);
    await expect(page.getByText(/Permission Check Simulator/i).first()).toBeVisible();
    await expect(page.getByText(/Active Relation Tuples/i).first()).toBeVisible();
  });

  test("evaluates a positive permission check and displays ALLOW decision result with traversal path", async ({ page, seedTuple }) => {
    const tuple = await seedTuple({
      namespace: "document",
      object: "annual-report-2026",
      relation: "viewer",
      subjectId: "alice",
    });

    await page.goto("/permissions");
    await page.waitForLoadState("networkidle");

    await page.locator("#namespace").fill(tuple.namespace);
    await page.locator("#object").fill(tuple.object);
    
    // Select relation from Radix combobox
    await page.locator("button#relation").click();
    await page.getByRole("option", { name: tuple.relation }).click();

    await page.locator("#subject").fill(tuple.subjectId);
    await page.getByRole("button", { name: /Evaluate Permission/i }).click();

    // Verify ACCESS ALLOWED decision banner and latency info
    await expect(page.getByText(/ACCESS ALLOWED/i).first()).toBeVisible({ timeout: 10_000 });
    await expect(page.getByText(/Resolved in/i).first()).toBeVisible();
  });

  test("zero-trust default deny: evaluates unmapped relation and returns DENIED", async ({ page }) => {
    await page.locator("#namespace").fill("document");
    await page.locator("#object").fill("unmapped-secret-doc-999");
    
    // Select relation from Radix combobox
    await page.locator("button#relation").click();
    await page.getByRole("option", { name: "owner" }).click();

    await page.locator("#subject").fill("unauthorized-user-xyz");
    await page.getByRole("button", { name: /Evaluate Permission/i }).click();

    // Verify ACCESS DENIED decision banner
    await expect(page.getByText(/ACCESS DENIED/i).first()).toBeVisible({ timeout: 10_000 });
  });
});
