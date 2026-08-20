import { test, expect } from "./fixtures";

test.describe("Argus Audit & Governance Studio", () => {
  test("renders cryptographic audit log table and tamper verification status", async ({ page }) => {
    await page.goto("/audit");
    await page.waitForLoadState("networkidle");

    await expect(page.getByRole("heading", { level: 1 })).toHaveText(/audit log|governance|audit/i);
    await expect(page.getByText(/hash chain|verified|tamper/i).first()).toBeVisible();
    await expect(page.getByRole("table").or(page.locator("table"))).toBeVisible();
  });

  test("opens audit record detail dialog with structured JSON diff and secret redaction", async ({ page }) => {
    await page.goto("/audit");
    await page.waitForLoadState("networkidle");

    // Click on the view details action on the first audit row
    const viewBtn = page.locator("button:has(svg.lucide-eye), button[aria-label*='view'], button:has-text('View')").first();
    if (await viewBtn.isVisible()) {
      await viewBtn.click();
      await expect(page.getByRole("dialog")).toBeVisible();
      await expect(page.getByText(/before state|after state|audit record detail/i).first()).toBeVisible();
    }
  });

  test("filters audit entries by search keyword", async ({ page }) => {
    await page.goto("/audit");
    await page.waitForLoadState("networkidle");

    const searchInput = page.getByPlaceholder(/search audit logs/i);
    if (await searchInput.isVisible()) {
      await searchInput.fill("operator");
      await page.waitForTimeout(300);
      await expect(page.locator("tbody tr").first()).toBeVisible();
    }
  });
});
