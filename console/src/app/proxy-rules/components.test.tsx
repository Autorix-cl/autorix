import { describe, it, expect, vi, beforeEach } from "vitest";
import React from "react";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { RuleBuilderSheet } from "./rule-builder-sheet";
import { VersionsDialog } from "./versions-dialog";

vi.mock("@/lib/api/schema", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/api/schema")>();
  return {
    ...actual,
    fetchAndParse: vi.fn(),
  };
});

import { fetchAndParse } from "@/lib/api/schema";
import { I18nProvider } from "@/lib/i18n";

function renderWithClient(ui: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <I18nProvider>
      <QueryClientProvider client={queryClient}>
        {ui}
      </QueryClientProvider>
    </I18nProvider>
  );
}


describe("RuleBuilderSheet", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders trigger button and opens wizard sheet", async () => {
    vi.mocked(fetchAndParse).mockResolvedValue({
      ok: true,
      data: {
        authenticators: [{ name: "jwt", description: "JWT", config_schema: {} }],
        authorizers: [{ name: "allow", description: "Allow", config_schema: {} }],
        mutators: [{ name: "header", description: "Header", config_schema: {} }],
      },
    });

    renderWithClient(<RuleBuilderSheet />);

    const triggerBtn = screen.getByRole("button", { name: /New Proxy Rule/i });
    expect(triggerBtn).toBeDefined();

    fireEvent.click(triggerBtn);

    expect(screen.getByText(/Pipeline Rule Wizard/i)).toBeDefined();
    expect(screen.getByLabelText(/Rule ID/i)).toBeDefined();
    expect(screen.getByLabelText(/URL Pattern/i)).toBeDefined();
  });
});

describe("VersionsDialog", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders trigger button and displays version list", async () => {
    vi.mocked(fetchAndParse).mockResolvedValue({
      ok: true,
      data: [
        {
          version: 1,
          description: "Initial snapshot",
          rules: [
            {
              id: "rule-1",
              match: { url: "/admin/<.*>", methods: ["GET"] },
              authenticators: [],
              authorizer: { handler: "allow" },
              mutators: [],
              upstream: { url: "http://upstream:8080" },
            },
          ],
          created_at: "2026-09-02T12:00:00Z",
        },
      ],
    });

    renderWithClient(<VersionsDialog />);

    const triggerBtn = screen.getByRole("button", { name: /Version History/i });
    expect(triggerBtn).toBeDefined();

    fireEvent.click(triggerBtn);

    await waitFor(() => {
      expect(screen.getByText(/Rule Set Snapshots & Rollback/i)).toBeDefined();
      expect(screen.getByText(/Version 1/i)).toBeDefined();
      expect(screen.getByText(/Initial snapshot/i)).toBeDefined();
    });
  });
});

