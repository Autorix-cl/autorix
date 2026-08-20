import { test, expect } from "./fixtures";

test.describe("Argus Operator RBAC & Directory", () => {
  test("renders operators directory with master break-glass administrator", async ({ page }) => {
    await page.goto("/operators");
    await page.waitForLoadState("networkidle");

    await expect(page.getByRole("heading", { level: 1 })).toHaveText(/console operators|operators/i);
    await expect(page.getByText(/break-glass sovereignty policy|break-glass/i).first()).toBeVisible();
    await expect(page.locator("main").getByText("admin@autorix.local")).toBeVisible();
  });
});
