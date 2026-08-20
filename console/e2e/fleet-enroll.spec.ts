import { test, expect } from "./fixtures";

test.describe("Argus Fleet Engine Enrollment Wizard", () => {
  test("mints enrollment token through wizard and displays multi-runtime setup instructions", async ({ page }) => {
    await page.goto("/fleet/enroll");
    await page.waitForLoadState("networkidle");

    await expect(page.getByRole("heading", { level: 1 })).toHaveText(/enroll iam engine|enroll engine/i);

    // Step 1: Click Continue
    await page.getByRole("button", { name: /continue/i }).click();

    // Step 2: Click Mint Enrollment Token
    await page.getByRole("button", { name: /mint enrollment token/i }).click();

    // Verify token and runtime tabs appear
    await expect(page.getByText(/enrollment token minted/i)).toBeVisible({ timeout: 10_000 });
    await expect(page.getByText(/aet_live_/i).first()).toBeVisible();
    await expect(page.getByRole("tab", { name: /docker/i })).toBeVisible();
  });
});
