import { test, expect } from "./fixtures";

test.describe("Cross-Engine Explorer (P6-S8)", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/explorer");
    await page.waitForLoadState("networkidle");
  });

  test("renders cross-engine explorer and tabs", async ({ page }) => {
    await expect(page.getByRole("heading", { level: 1 })).toContainText(/cross-engine/i);
    await expect(page.getByRole("button", { name: /unified subject/i })).toBeVisible();
    await expect(page.getByRole("button", { name: /effective access/i })).toBeVisible();
    await expect(page.getByRole("button", { name: /request simulator/i })).toBeVisible();
    await expect(page.getByRole("button", { name: /consistency checks/i })).toBeVisible();
  });

  test("switches between explorer views", async ({ page }) => {
    // Switch to Effective Access Explorer
    await page.getByRole("button", { name: /effective access/i }).click();
    await expect(page.getByRole("button", { name: /^evaluate$/i })).toBeVisible();

    // Switch to Request Simulator
    await page.getByRole("button", { name: /request simulator/i }).click();
    await expect(page.getByRole("button", { name: /simulate request/i })).toBeVisible();

    // Switch to Consistency Checks
    await page.getByRole("button", { name: /consistency checks/i }).click();
    await expect(page.getByRole("button", { name: /re-scan/i })).toBeVisible();
  });
});
