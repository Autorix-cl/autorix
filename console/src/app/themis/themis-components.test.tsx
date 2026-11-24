import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import React from "react";
import { PolicyTestSuiteDialog } from "./policy-test-suite-dialog";
import { PolicyVersionsDialog } from "./policy-versions-dialog";
import { DryRunPlayground } from "./dry-run-playground";

vi.mock("@/lib/auth/use-permission", () => ({
  usePermission: () => ({
    operator: { id: "op-1", email: "admin@example.com", name: "Admin", role: "owner", is_local: true, is_active: true },
    isLoading: false,
    can: () => true,
    hasRole: () => true,
    checkSession: vi.fn().mockResolvedValue(true),
  }),
  Can: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

describe("Themis Depth Components", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    global.fetch = vi.fn();
  });

  it("renders PolicyVersionsDialog and triggers rollback", async () => {
    const handleRolledBack = vi.fn();

    // 1. Load versions
    vi.mocked(global.fetch).mockResolvedValueOnce(
      new Response(
        JSON.stringify([
          {
            id: "pv_1",
            policy_id: "pol_1",
            version: 1,
            name: "Initial Snapshot",
            expression: "request.role == 'admin'",
            created_at: "2026-09-02T10:00:00Z",
          },
        ]),
        { status: 200, headers: { "Content-Type": "application/json" } },
      ),
    );

    render(
      <PolicyVersionsDialog
        policyId="pol_1"
        policyName="Admin Rule"
        isOpen={true}
        onOpenChange={vi.fn()}
        onRolledBack={handleRolledBack}
      />,
    );

    await waitFor(() => {
      expect(screen.getByText("Initial Snapshot")).toBeDefined();
      expect(screen.getByText("v1")).toBeDefined();
    });

    // 2. Click rollback
    vi.mocked(global.fetch).mockResolvedValueOnce(
      new Response(JSON.stringify({ status: "success" }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );

    const rollbackBtn = screen.getByRole("button", { name: /^Rollback$/i });
    fireEvent.click(rollbackBtn);

    // Confirm rollback inside ConfirmDialog
    const confirmRollbackBtn = await screen.findByRole("button", { name: /Rollback to v1/i });
    fireEvent.click(confirmRollbackBtn);

    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith(
        "/api/themis/policies/pol_1/rollback/1",
        expect.objectContaining({ method: "POST" }),
      );
      expect(handleRolledBack).toHaveBeenCalled();
    });
  });

  it("renders PolicyTestSuiteDialog and executes test suite", async () => {
    // 1. Load fixtures
    vi.mocked(global.fetch).mockResolvedValueOnce(
      new Response(
        JSON.stringify([
          {
            id: "fix_1",
            policy_id: "pol_1",
            name: "Admin MFA fixture",
            expected_result: true,
            payload: { role: "admin", mfa: true },
          },
        ]),
        { status: 200, headers: { "Content-Type": "application/json" } },
      ),
    );

    render(<PolicyTestSuiteDialog policyId="pol_1" policyName="Admin Rule" isOpen={true} onOpenChange={vi.fn()} />);

    await waitFor(() => {
      expect(screen.getByText("Admin MFA fixture")).toBeDefined();
      expect(screen.getByText("Expected: Allow")).toBeDefined();
    });

    // 2. Run test suite
    vi.mocked(global.fetch).mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          policy_id: "pol_1",
          all_passed: true,
          total_tests: 1,
          passed_tests: 1,
          failed_tests: 0,
          results: [
            {
              fixture_id: "fix_1",
              fixture_name: "Admin MFA fixture",
              expected_result: true,
              actual_result: true,
              passed: true,
              error: "",
            },
          ],
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      ),
    );

    const runBtn = screen.getByRole("button", { name: /Run All Tests/i });
    fireEvent.click(runBtn);

    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith(
        "/api/themis/policies/pol_1/test-suite",
        expect.objectContaining({ method: "POST" }),
      );
      expect(screen.getByText("All Test Scenarios Passed")).toBeDefined();
    });
  });

  it("validates scratchpad expression in DryRunPlayground", async () => {
    render(<DryRunPlayground />);

    const scratchpadTab = screen.getByText("Expression Scratchpad");
    fireEvent.click(scratchpadTab);

    expect(screen.getByText("CEL Expression Scratchpad")).toBeDefined();
  });
});
