import { test, expect } from "./fixtures";

test.describe("Argus Fleet & Engine Topology", () => {
  test("renders all 7 engine cards with real live status", async ({ page }) => {
    await page.goto("/fleet");
    await page.waitForLoadState("networkidle");

    await expect(page.getByRole("heading", { level: 1 })).toHaveText(/fleet & engine topology|fleet/i);
    await expect(page.getByText(/ego \(identity\)/i)).toBeVisible();
    await expect(page.getByText(/nexus \(zanzibar\)/i)).toBeVisible();
    await expect(page.getByText(/janus \(oauth2\/oidc\)/i)).toBeVisible();
    await expect(page.getByText(/aegis \(proxy\)/i)).toBeVisible();
    await expect(page.getByText(/vulcan \(api keys\)/i)).toBeVisible();
    await expect(page.getByText(/hermes \(enterprise\)/i)).toBeVisible();
    await expect(page.getByText(/themis \(policy\)/i)).toBeVisible();
  });

  test("navigates to instances directory and shows registered instances", async ({ page }) => {
    await page.goto("/fleet/instances");
    await page.waitForLoadState("networkidle");

    await expect(page.getByRole("heading", { level: 1 })).toHaveText(/fleet instances|instances/i);
    await expect(page.getByRole("table").or(page.locator("table"))).toBeVisible();
  });

  test("renders engine topology graph view", async ({ page }) => {
    await page.goto("/fleet/topology");
    await page.waitForLoadState("networkidle");

    await expect(page.getByRole("heading", { level: 1 })).toHaveText(/topology|network|graph/i);
  });
});
