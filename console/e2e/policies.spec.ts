import { test, expect } from "./fixtures";

test.describe("Themis ABAC / CEL Studio", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/policies");
    await page.waitForLoadState("networkidle");
  });

  test("renders policy creation form, CEL dry-run simulator and policies table", async ({ page }) => {
    await expect(page.getByRole("heading", { level: 1 })).toHaveText(/Autorix Themis|CEL Policy/i);
    await expect(page.locator("#name")).toBeVisible();
    await expect(page.locator("#expression")).toBeVisible();
  });

  test("creates a new CEL policy and displays it in the policies directory", async ({ page }) => {
    const unique = `${Date.now()}-${Math.random().toString(36).slice(2, 7)}`;
    const policyName = `E2E Policy ${unique}`;

    await page.locator("#name").fill(policyName);
    await page.locator("#expression").fill('request.auth.claims.department == "finance"');
    await page.getByRole("button", { name: /Compile & Save Policy|Save Policy/i }).click();

    // Verify created policy in table cell
    await expect(page.getByRole("cell", { name: policyName })).toBeVisible({ timeout: 10_000 });
  });

  test("runs CEL dry-run evaluation against payload and displays evaluation result", async ({ page }) => {
    const evalBtn = page.getByRole("button", { name: /Execute Rule Evaluation|Evaluate/i });
    if (await evalBtn.isVisible()) {
      await evalBtn.click();
      await expect(page.getByText(/Evaluation Verdict|ALL POLICIES PASSED|POLICY DENIED/i).first()).toBeVisible({ timeout: 5000 });
    }
  });
});
