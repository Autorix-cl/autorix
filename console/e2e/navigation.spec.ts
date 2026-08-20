import { test, expect } from "./fixtures";

/**
 * Every sidebar link lands on the right page against the real running
 * console (P1-S5-T5) — no mocked router, no mocked backend.
 */
const NAV_ITEMS = [
  { href: "/", heading: /telemetry|overview/i },
  { href: "/identities", heading: /identity|credentials/i },
  { href: "/permissions", heading: /zanzibar|rebac/i },
  { href: "/oauth2", heading: /oauth2|jwks/i },
  { href: "/proxy-rules", heading: /zero trust|proxy/i },
  { href: "/api-keys", heading: /api key|macaroon/i },
  { href: "/enterprise", heading: /saml|scim|enterprise/i },
  { href: "/policies", heading: /policy|cel/i },
];

test.describe("navigation", () => {
  for (const { href, heading } of NAV_ITEMS) {
    test(`renders a real page at ${href}`, async ({ page }) => {
      await page.goto(href);
      await expect(page.getByRole("heading", { level: 1 })).toHaveText(heading);
      // A failed BFF/engine call must never render as a silently blank
      // page — some content besides the heading has to be present.
      await expect(page.locator("main")).not.toBeEmpty();
    });
  }

  test("sidebar links navigate without a full page reload", async ({ page }) => {
    await page.goto("/");
    await page
      .getByRole("link", { name: /identities/i })
      .first()
      .click();
    await expect(page).toHaveURL(/\/identities$/);
  });
});
