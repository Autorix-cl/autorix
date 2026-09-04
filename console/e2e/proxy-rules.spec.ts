import { test, expect } from "./fixtures";

test.describe("Aegis Zero-Trust Proxy Studio (Ory Oathkeeper PEP)", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/proxy-rules");
    await page.waitForLoadState("networkidle");
  });

  test("renders proxy rules and rule matching simulator", async ({ page }) => {
    await expect(page.getByRole("heading", { level: 1 })).toHaveText(/Autorix Aegis|Zero Trust/i);
    await expect(page.getByText(/Proxy Routing & Enforcement Simulator/i).first()).toBeVisible();
    await expect(page.getByText(/Declarative Rules Configuration/i).first()).toBeVisible();
  });

  test("runs 3-stage test match evaluation for a request URI against active rules", async ({ page, seedProxyRule }) => {
    const rule = await seedProxyRule();

    await page.goto("/proxy-rules");
    await page.waitForLoadState("networkidle");

    // Switch to Traffic Simulator tab
    await page.getByRole("tab", { name: /Simulator/i }).click();

    const pathInput = page.locator("input[value*='/api/v1'], input[placeholder*='api/v1']").first();
    const testBtn = page.getByRole("button", { name: /Simulate Match/i });

    await pathInput.fill(rule.match_path.replace("/<.*>", "/item-123"));
    await testBtn.click();

    // Verify 3 pipeline stages are displayed: Authenticator, Authorizer, Mutator
    await expect(page.getByText(/Rule Matched/i).first()).toBeVisible({ timeout: 5000 });
    await expect(page.getByText(/1\. Authenticator/i)).toBeVisible();
    await expect(page.getByText(/2\. Authorizer/i)).toBeVisible();
    await expect(page.getByText(/3\. Mutator/i)).toBeVisible();
  });

  test("zero-trust default deny: returns no-match banner on unmapped URI", async ({ page }) => {
    // Switch to Traffic Simulator tab
    await page.getByRole("tab", { name: /Simulator/i }).click();

    const pathInput = page.locator("input[value*='/api/v1'], input[placeholder*='api/v1']").first();
    const testBtn = page.getByRole("button", { name: /Simulate Match/i });

    await pathInput.fill("/completely/unmapped/unknown/secret/resource/999");
    await testBtn.click();

    // Verify Default Deny banner
    await expect(page.getByText(/No rule matched this path/i).first()).toBeVisible({ timeout: 5000 });
  });

  test("switches between studio tabs and renders scrollable declarative manifest", async ({ page }) => {
    // Check initial tab is Routing Pipeline
    await expect(page.getByText(/Active Proxy Rules & Order/i)).toBeVisible();

    // Capture screenshot of Rules Pipeline
    await page.screenshot({
      path: "/Users/macbook/.gemini/antigravity-cli/brain/3a3fb493-f34f-4483-8b1d-d2f8068933b2/screenshots/cloud-aegis-rules.png",
    });

    // Switch to Traffic Simulator tab
    await page.getByRole("tab", { name: /Simulator/i }).click();
    await expect(page.getByRole("heading", { name: "Pipeline Request Matcher" })).toBeVisible();

    // Capture screenshot of Simulator tab
    await page.screenshot({
      path: "/Users/macbook/.gemini/antigravity-cli/brain/3a3fb493-f34f-4483-8b1d-d2f8068933b2/screenshots/cloud-aegis-simulator.png",
    });

    // Switch to Declarative Manifest tab
    await page.getByRole("tab", { name: /Declarative Rules Configuration|Manifest/i }).click();
    await expect(page.getByText("GET /rules (Aegis admin API, live)")).toBeVisible();

    // Capture screenshot of Declarative Manifest tab
    await page.screenshot({
      path: "/Users/macbook/.gemini/antigravity-cli/brain/3a3fb493-f34f-4483-8b1d-d2f8068933b2/screenshots/cloud-aegis-manifest.png",
    });
  });
});
