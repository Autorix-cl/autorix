import { test, expect } from "./fixtures";

test.describe("Themis ABAC / CEL Studio", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/policies");
    await page.waitForLoadState("networkidle");
  });

  test("renders policy creation form, CEL dry-run simulator and policies table", async ({ page }) => {
    await expect(page.getByRole("heading", { level: 1 })).toHaveText(/Autorix Themis|CEL Policy/i);
    await expect(page.getByRole("button", { name: /Create ABAC \/ CEL Policy/i })).toBeVisible();
    await expect(page.getByRole("button", { name: /Execute Rule Evaluation/i })).toBeVisible();
  });

  test("creates a new CEL policy and displays it in the policies directory", async ({ page }) => {
    const unique = `${Date.now()}-${Math.random().toString(36).slice(2, 7)}`;
    const policyName = `E2E Policy ${unique}`;

    await page.getByRole("button", { name: /Create ABAC \/ CEL Policy/i }).click();
    await page.locator("#name").fill(policyName);
    await page.locator("#expression").fill('request.auth.claims.department == "finance"');
    await page.getByRole("button", { name: /Compile & Save Policy/i }).click();
    await expect(page.getByRole("dialog")).toBeHidden();

    // Filter by the unique policy name to account for multi-page directories
    await page.getByPlaceholder(/Filter policies/i).fill(unique);

    // Verify created policy in table cell
    await expect(page.getByRole("cell", { name: policyName })).toBeVisible({ timeout: 10_000 });
  });

  test("runs CEL dry-run evaluation against payload and displays evaluation result", async ({ page }) => {
    const evalBtn = page.getByRole("button", { name: /Execute Rule Evaluation/i });
    await evalBtn.click();
    await expect(page.getByText(/Evaluation Verdict/i)).toBeVisible({ timeout: 10_000 });
  });
});
