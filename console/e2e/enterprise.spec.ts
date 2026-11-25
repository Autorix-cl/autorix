import { execFileSync } from "node:child_process";
import { test, expect } from "./fixtures";

function generateTestCertificatePem(): string {
  return execFileSync(
    "openssl",
    [
      "req",
      "-x509",
      "-newkey",
      "ec",
      "-pkeyopt",
      "ec_paramgen_curve:prime256v1",
      "-keyout",
      "/dev/null",
      "-nodes",
      "-days",
      "365",
      "-subj",
      "/CN=e2e-test-idp.example.com",
    ],
    { encoding: "utf8", stdio: ["ignore", "pipe", "ignore"] },
  ).trim();
}

test.describe("Hermes SAML & SCIM Enterprise Studio", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/enterprise");
    await page.waitForLoadState("load");
  });

  test("renders SAML SP metadata XML and SCIM directory overview across studio tabs", async ({ page }, testInfo) => {
    await page.setViewportSize({ width: 1440, height: 1100 });
    await expect(page.getByRole("heading", { level: 1 })).toHaveText(/Autorix Hermes|Enterprise/i);

    // Tab 1: SAML Identity Providers
    await expect(page.getByRole("tab", { name: /SAML Identity Providers/i })).toBeVisible();
    await expect(page.getByText(/Active IdP Connections & Assertion Mappings/i)).toBeVisible();
    await page.screenshot({
      path: testInfo.outputPath("cloud-hermes-providers.png"),
      fullPage: true,
    });

    // Tab 2: SCIM Directory Sync
    const scimTab = page.getByRole("tab", { name: /SCIM Directory Sync/i });
    await expect(async () => {
      await scimTab.click();
      await expect(scimTab).toHaveAttribute("aria-selected", "true", { timeout: 1000 });
    }).toPass({ timeout: 15_000 });
    await expect(page.getByText(/SCIM 2.0 Directory Management & Sync Monitor/i).first()).toBeVisible({
      timeout: 10_000,
    });
    await page.screenshot({
      path: testInfo.outputPath("cloud-hermes-scim.png"),
      fullPage: true,
    });

    // Tab 3: SP Metadata & Endpoints
    const metadataTab = page.getByRole("tab", { name: /SP Metadata & Endpoints/i });
    await expect(async () => {
      await metadataTab.click();
      await expect(metadataTab).toHaveAttribute("aria-selected", "true", { timeout: 1000 });
    }).toPass({ timeout: 15_000 });
    await expect(page.getByText(/Service Provider \(SP\) Metadata XML/i).first()).toBeVisible({ timeout: 10_000 });
    await expect(page.getByText(/LIVE SAML SP METADATA/i)).toBeVisible({ timeout: 15_000 });
    await expect(page.getByText("http://localhost:4477/saml/acs", { exact: true })).toBeVisible({ timeout: 10_000 });
    await page.screenshot({
      path: testInfo.outputPath("cloud-hermes-metadata.png"),
      fullPage: true,
    });
  });

  test("registers a new enterprise SAML 2.0 identity provider and verifies success", async ({ page }) => {
    const unique = `${Date.now()}-${Math.random().toString(36).slice(2, 7)}`;
    const providerSlug = `okta-${unique}`;
    const providerName = `Okta Production IdP ${unique}`;
    const certPem = generateTestCertificatePem();

    await page.getByRole("button", { name: /New Connection Wizard/i }).click();
    await expect(page.getByRole("heading", { name: /SAML Connection Setup Wizard/i })).toBeVisible();

    await page.getByPlaceholder("e.g. okta-corporate").fill(providerSlug);
    await page.getByPlaceholder("e.g. Okta Corporate").fill(providerName);
    await page
      .getByPlaceholder("https://company.okta.com/app/autorix/sso/saml")
      .fill(`https://auth.company-${unique}.okta.com/app/sso/saml`);
    await page.locator("#saml-cert-pem").fill(certPem);

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
