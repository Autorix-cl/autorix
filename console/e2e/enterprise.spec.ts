import { test, expect } from "./fixtures";

test.describe("Hermes SAML & SCIM Enterprise Studio", () => {
  test("renders SAML provider registration, SCIM directory, and SP metadata XML", async ({ page }) => {
    await page.goto("/enterprise");
    await page.waitForLoadState("networkidle");

    await expect(page.getByRole("heading", { level: 1 })).toHaveText(/enterprise|saml|scim|hermes/i);
    await expect(page.getByText(/register saml provider|scim 2.0/i).first()).toBeVisible();
    await expect(page.getByText(/sp metadata xml|metadata/i).first()).toBeVisible();
  });
});
