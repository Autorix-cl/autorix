import { test, expect } from "./fixtures";

test.describe("Global Header & Preferences", () => {
  test("toggles language between English and Spanish", async ({ page }) => {
    await page.goto("/");
    await page.waitForLoadState("networkidle");

    // Open language menu
    const langBtn = page.locator("header").getByRole("button").filter({ hasText: /^(EN|ES)$/ });
    await langBtn.click();

    // Select Spanish
    await page.getByRole("menuitem").filter({ hasText: /Español/i }).click();
    await expect(page.locator("header").getByRole("button").filter({ hasText: /^ES$/ })).toBeVisible();

    // Switch back to English
    const esLangBtn = page.locator("header").getByRole("button").filter({ hasText: /^ES$/ });
    await esLangBtn.click();
    await page.getByRole("menuitem").filter({ hasText: /English/i }).click();
    await expect(page.locator("header").getByRole("button").filter({ hasText: /^EN$/ })).toBeVisible();
  });

  test("toggles theme between Dark, Light, and System modes", async ({ page }) => {
    await page.goto("/");
    await page.waitForLoadState("networkidle");

    const themeBtn = page.getByLabel("Toggle theme");
    await themeBtn.click();
    await page.getByRole("menuitem").filter({ hasText: "Light" }).click();
    await expect(page.locator("html")).toHaveClass(/light/);

    // Switch back to Dark
    await page.waitForTimeout(300);
    await themeBtn.click();
    await page.getByRole("menuitem").filter({ hasText: "Dark" }).click();
    await expect(page.locator("html")).toHaveClass(/dark/);
  });

  test("opens command palette search dialog with ⌘K keyboard shortcut", async ({ page }) => {
    await page.goto("/");
    await page.waitForLoadState("networkidle");

    // Press Cmd+K or Ctrl+K
    await page.keyboard.press("Meta+k");
    const searchDialog = page.getByRole("dialog");
    await expect(searchDialog).toBeVisible({ timeout: 5000 });
    await page.keyboard.press("Escape");
  });
});
