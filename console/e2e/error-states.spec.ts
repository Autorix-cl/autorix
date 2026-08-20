import { test, expect } from "./fixtures";

test.describe("Error & Edge States", () => {
  test("renders 404 not-found page on non-existent route", async ({ page }) => {
    await page.goto("/non-existent-random-route-404");
    await expect(page.getByText(/404|not found|page not found/i).first()).toBeVisible({ timeout: 5000 });
  });

  test("rejects mutating request without valid CSRF token with 403 Forbidden", async ({ request }) => {
    const res = await request.post("/api/identities", {
      data: { email: "attacker@malicious.com" },
      headers: { "X-CSRF-Token": "invalid-forged-csrf-token" },
    });

    expect(res.status()).toBe(403);
  });
});
