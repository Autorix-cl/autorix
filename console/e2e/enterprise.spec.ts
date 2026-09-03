import { test, expect } from "./fixtures";

test.describe("Hermes SAML & SCIM Enterprise Studio", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/enterprise");
    await page.waitForLoadState("networkidle");
  });

  test("renders SAML SP metadata XML and SCIM directory overview", async ({ page }) => {
    await expect(page.getByRole("heading", { level: 1 })).toHaveText(/Autorix Hermes|Enterprise/i);
    await expect(page.getByText(/Service Provider \(SP\) Metadata XML/i).first()).toBeVisible();
    await expect(page.getByText(/SCIM 2.0 Directory Management & Sync Monitor/i).first()).toBeVisible();
  });

  test("registers a new enterprise SAML 2.0 identity provider and verifies success", async ({ page }) => {
    const unique = `${Date.now()}-${Math.random().toString(36).slice(2, 7)}`;
    const providerSlug = `okta-${unique}`;
    const providerName = `Okta Production IdP ${unique}`;

    await page.getByRole("button", { name: /New Connection Wizard/i }).click();
    await expect(page.getByRole("heading", { name: /SAML Connection Setup Wizard/i })).toBeVisible();

    await page.getByPlaceholder("e.g. okta-corporate").fill(providerSlug);
    await page.getByPlaceholder("e.g. Okta Corporate").fill(providerName);
    await page.getByPlaceholder("https://company.okta.com/app/autorix/sso/saml").fill(`https://auth.company-${unique}.okta.com/app/sso/saml`);

    // Step 1 -> Step 2
    await page.getByRole("button", { name: /^Next/i }).click();

    // Step 2 -> Step 3
    await page.getByRole("button", { name: /^Next/i }).click();

    // Step 3 -> Register
    await page.getByRole("button", { name: /Register & Verify Connection/i }).click();

    // Verify success banner appears in wizard
    await expect(page.getByText(/Connection established successfully/i)).toBeVisible({ timeout: 10_000 });
  });
});
