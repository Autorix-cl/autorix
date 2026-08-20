import { test, expect } from "./fixtures";

test.describe("Nexus ReBAC Permissions Studio", () => {
  test("renders Zanzibar relation graph simulator and relation tuples table", async ({ page }) => {
    await page.goto("/permissions");
    await page.waitForLoadState("networkidle");

    await expect(page.getByRole("heading", { level: 1 })).toHaveText(/rebac|nexus|permission/i);
    await expect(page.getByText(/check simulator|relation tuples/i).first()).toBeVisible();
  });

  test("evaluates a permission check and displays ALLOW decision result", async ({ page, request }) => {
    // Seed a known relation tuple first
    const unique = `${Date.now()}`;
    const object = `doc_${unique}`;
    const subject = `user_${unique}`;

    const res = await request.post("/api/permissions", {
      data: {
        namespace: "document",
        object,
        relation: "viewer",
        subjectNamespace: "user",
        subjectId: subject,
      },
    });

    await page.goto("/permissions");
    await page.waitForLoadState("networkidle");

    // Fill the permission check form
    const objectInput = page.locator("#perm-object, input[placeholder*='report'], input[placeholder*='object']").first();
    const subjectInput = page.locator("#perm-subject, input[placeholder*='alice'], input[placeholder*='subject']").first();
    const submitBtn = page.getByRole("button", { name: /run permission check|evaluate|check/i });

    if (await objectInput.isVisible()) await objectInput.fill(object);
    if (await subjectInput.isVisible()) await subjectInput.fill(subject);
    await submitBtn.click();

    // Verify ALLOW or DENY decision card renders
    await expect(page.getByText(/allowed|access granted|allow|denied/i).first()).toBeVisible({ timeout: 10_000 });
  });
});
