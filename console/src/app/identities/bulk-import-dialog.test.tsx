import { describe, it, expect, vi, beforeEach } from "vitest";
import React from "react";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { BulkImportDialog } from "./bulk-import-dialog";

describe("BulkImportDialog", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    global.fetch = vi.fn();
  });

  it("renders textarea and dry run button", () => {
    render(<BulkImportDialog isOpen={true} onOpenChange={vi.fn()} />);

    expect(screen.getByText("Bulk Import Identities")).toBeDefined();
    expect(screen.getByRole("button", { name: /Dry-Run Validation/i })).toBeDefined();
    expect(screen.getByRole("button", { name: /Commit Import/i })).toBeDefined();
  });

  it("executes dry-run validation and displays row summary", async () => {
    vi.mocked(global.fetch).mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          total: 2,
          succeeded: 1,
          failed: 1,
          errors: [{ row: 2, email: "invalid", error: "Missing or invalid email address" }],
          dry_run: true,
        }),
        { status: 200, headers: { "Content-Type": "application/json" } }
      )
    );

    render(<BulkImportDialog isOpen={true} onOpenChange={vi.fn()} />);

    const textarea = screen.getByPlaceholderText(/user@example.com/i);
    fireEvent.change(textarea, {
      target: { value: "alice@autorix.io,Alice,Smith\ninvalid,Bob,Jones" },
    });

    const dryRunBtn = screen.getByRole("button", { name: /Dry-Run Validation/i });
    fireEvent.click(dryRunBtn);

    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith(
        "/api/identities/bulk-import",
        expect.objectContaining({ method: "POST" })
      );
      expect(screen.getByText(/1 Valid/i)).toBeDefined();
      expect(screen.getByText(/1 Errors/i)).toBeDefined();
      expect(screen.getByText(/Missing or invalid email address/i)).toBeDefined();
    });
  });

  it("commits valid rows and calls onSuccess callback", async () => {
    const handleSuccess = vi.fn();
    const handleOpenChange = vi.fn();

    vi.mocked(global.fetch).mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          total: 1,
          succeeded: 1,
          failed: 0,
          errors: [],
          dry_run: false,
        }),
        { status: 200, headers: { "Content-Type": "application/json" } }
      )
    );

    render(<BulkImportDialog isOpen={true} onOpenChange={handleOpenChange} onSuccess={handleSuccess} />);

    const textarea = screen.getByPlaceholderText(/user@example.com/i);
    fireEvent.change(textarea, {
      target: { value: "alice@autorix.io,Alice,Smith" },
    });

    const commitBtn = screen.getByRole("button", { name: /Commit Import/i });
    fireEvent.click(commitBtn);

    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith(
        "/api/identities/bulk-import",
        expect.objectContaining({
          method: "POST",
          body: expect.stringContaining('"dry_run":false'),
        })
      );
      expect(handleSuccess).toHaveBeenCalled();
      expect(handleOpenChange).toHaveBeenCalledWith(false);
    });
  });
});
