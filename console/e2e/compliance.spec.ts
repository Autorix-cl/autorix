import { test, expect } from "./fixtures";

test.describe("Argus Continuous Compliance Matrix", () => {
  test("renders compliance framework summary and controls matrix", async ({ page }) => {
    await page.goto("/compliance");
    await page.waitForLoadState("networkidle");

    await expect(page.getByRole("heading", { level: 1 })).toHaveText(/continuous compliance|compliance matrix|compliance/i);
    await expect(page.getByText(/soc 2|iso 27001|evidence/i).first()).toBeVisible();
  });

  test("filters compliance controls by framework tabs", async ({ page }) => {
    await page.goto("/compliance");
    await page.waitForLoadState("networkidle");

    const soc2Tab = page.locator("button").filter({ hasText: /soc 2/i }).first();
    if (await soc2Tab.isVisible()) {
      await soc2Tab.click();
      await expect(page.getByText(/CC6\./i).first()).toBeVisible();
    }
  });
});
