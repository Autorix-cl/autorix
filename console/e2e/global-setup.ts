import { chromium, type FullConfig } from "@playwright/test";
import * as fs from "fs";
import * as path from "path";

async function globalSetup(config: FullConfig) {
  const baseURL = config.projects[0]?.use?.baseURL || "http://localhost:3000";
  const authDir = path.join(__dirname, ".auth");
  const authFile = path.join(authDir, "operator.json");

  if (!fs.existsSync(authDir)) {
    fs.mkdirSync(authDir, { recursive: true });
  }

  const browser = await chromium.launch();
  const context = await browser.newContext({ baseURL });
  const page = await context.newPage();

  const email = process.env.AUTORIX_OPERATOR_EMAIL || "admin@autorix.local";
  const password = process.env.AUTORIX_OPERATOR_PASSWORD || "SecretMasterKey#2026";

  try {
    // Check if cluster needs setup
    const statusRes = await context.request.get("/api/auth/status");
    if (statusRes.ok()) {
      const status = await statusRes.json();
      if (!status.bootstrapped && process.env.AUTORIX_BOOTSTRAP_TOKEN) {
        await context.request.post("/api/auth/setup", {
          data: {
            bootstrap_token: process.env.AUTORIX_BOOTSTRAP_TOKEN,
            name: "Autorix Master Admin",
            email,
            password,
          },
        });
      }
    }

    // Authenticate operator
    await page.goto("/login");
    await page.waitForLoadState("networkidle");

    // If already redirected to dashboard (e.g. valid cookie retained), save state directly
    if (!page.url().includes("/login")) {
      await context.addCookies([
        { name: "autorix_active_env", value: "prod", domain: "localhost", path: "/" },
      ]);
      await context.storageState({ path: authFile });
      await browser.close();
      return;
    }

    const emailInput = page.getByRole("textbox", { name: /email/i }).or(page.locator('input[type="email"]'));
    const passwordInput = page.locator('input[type="password"]');
    const submitButton = page.getByRole("button", { name: /authenticate|sign in|login/i });

    await emailInput.fill(email);
    await passwordInput.fill(password);
    await submitButton.click();

    // Wait for redirect away from /login
    await page.waitForURL((url) => !url.pathname.startsWith("/login"), { timeout: 15_000 });
    await context.addCookies([
      { name: "autorix_active_env", value: "prod", domain: "localhost", path: "/" },
    ]);
    // Save storage state containing cookies and localStorage
    await context.storageState({ path: authFile });
  } catch (err) {
    console.error("Global setup authentication failed:", err);
    throw err;
  } finally {
    await browser.close();
  }
}

export default globalSetup;
