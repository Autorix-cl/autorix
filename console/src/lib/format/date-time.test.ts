import { describe, it, expect } from "vitest";
import { formatDateTime, formatRelativeTime } from "./date-time";

describe("Locale-aware formatting (P4-S6-T4)", () => {
  it("formats dates according to locale and timezone", () => {
    const date = new Date("2026-08-18T12:00:00Z");
    const formattedEn = formatDateTime(date, "en", "UTC");
    const formattedEs = formatDateTime(date, "es", "UTC");

    expect(formattedEn).toContain("2026");
    expect(formattedEs).toContain("2026");
  });

  it("formats relative times for recent audit events", () => {
    const now = Date.now();
    const fiveMinAgo = new Date(now - 5 * 60 * 1000);
    const relEn = formatRelativeTime(fiveMinAgo, "en");
    const relEs = formatRelativeTime(fiveMinAgo, "es");

    expect(relEn).toMatch(/5 (minute|min)s? ago/i);
    expect(relEs).toMatch(/(hace 5 minutos|hace 5 min)/i);
  });
});
