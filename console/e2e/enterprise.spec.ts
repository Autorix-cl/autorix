import { test, expect } from "./fixtures";

test.describe("Hermes SAML & SCIM Enterprise Studio", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/enterprise");
    await page.waitForLoadState("networkidle");
  });

  test("renders SAML SP metadata XML and SCIM directory overview", async ({ page }) => {
    await expect(page.getByRole("heading", { level: 1 })).toHaveText(/Autorix Hermes|Enterprise/i);
    await expect(page.getByText(/Service Provider \(SP\) Metadata XML/i).first()).toBeVisible();
    await expect(page.getByText(/SCIM 2.0 Synchronized Directory/i).first()).toBeVisible();
  });

  test("registers a new enterprise SAML 2.0 identity provider and verifies success", async ({ page }) => {
    const unique = `${Date.now()}-${Math.random().toString(36).slice(2, 7)}`;
    const providerId = `okta-${unique}`;
    const providerName = `Okta Production IdP ${unique}`;

    await page.locator("#providerId").fill(providerId);
    await page.locator("#providerName").fill(providerName);
    await page.locator("#idpEntityId").fill(`https://auth.company-${unique}.okta.com`);
    await page.locator("#idpSsoUrl").fill(`https://auth.company-${unique}.okta.com/app/sso/saml`);

    await page.getByRole("button", { name: /Register IdP & Setup SP/i }).click();

    // Verify success toast/message or form reset
    await expect(page.locator("#providerId")).toHaveValue("", { timeout: 10_000 });
  });
});
