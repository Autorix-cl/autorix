import { test, expect } from "./fixtures";

test.describe("Observability & Operations (Phase 7)", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/observability");
    await page.waitForLoadState("networkidle");
  });

  test("renders fleet observability overview with RED metrics", async ({ page }) => {
    await expect(page.getByRole("heading", { level: 1 })).toHaveText(/fleet observability/i);
    await expect(page.getByText(/Fleet Throughput/i)).toBeVisible();
    await expect(page.getByText(/Error Rate/i)).toBeVisible();
    await expect(page.getByText(/p95 Latency/i)).toBeVisible();
    await expect(page.getByText(/Allow Ratio/i)).toBeVisible();
  });

  test("switches tabs and interacts with diagnostics troubleshooter", async ({ page }) => {
    // Switch to Diagnostics & Incident Support after hydration
    await expect(async () => {
      await page.getByRole("button", { name: /diagnostics & incident/i }).click();
      await expect(page.getByText(/Instance-to-Instance Live Probe/i)).toBeVisible({ timeout: 1000 });
    }).toPass();

    // Verify sub-tabs in diagnostics
    await expect(page.getByRole("button", { name: /connectivity troubleshooter/i })).toBeVisible();
    await expect(page.getByRole("button", { name: /config drift/i })).toBeVisible();
    await expect(page.getByRole("button", { name: /migration status/i })).toBeVisible();
    await expect(page.getByRole("button", { name: /change correlation timeline/i })).toBeVisible();

    // Run a live probe
    const execBtn = page.getByRole("button", { name: /execute probe/i });
    await expect(execBtn).toBeVisible();
    await execBtn.click();

    // Verify 4-stage breakdown appears
    await expect(page.getByText(/1\. DNS Lookup/i)).toBeVisible();
    await expect(page.getByText(/2\. TCP Connect/i)).toBeVisible();
    await expect(page.getByText(/3\. TLS Handshake/i)).toBeVisible();
    await expect(page.getByText(/4\. Application HTTP/i)).toBeVisible();
  });
});
