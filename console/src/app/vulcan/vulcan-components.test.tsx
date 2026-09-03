import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import * as React from "react";
import { ScopeCatalogSheet } from "./scope-catalog-sheet";
import { KeyRotateDialog } from "./key-rotate-dialog";
import { MacaroonInspector } from "./macaroon-inspector";

describe("Vulcan UI Components", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    global.fetch = vi.fn();
  });

  it("renders ScopeCatalogSheet and displays scopes", async () => {
    vi.mocked(global.fetch).mockResolvedValueOnce(
      new Response(
        JSON.stringify([
          { name: "billing:read", description: "Read billing records" },
          { name: "auth:admin", description: "Full admin privileges" },
        ]),
        { status: 200, headers: { "Content-Type": "application/json" } }
      )
    );

    render(<ScopeCatalogSheet isOpen={true} onOpenChange={() => {}} />);

    expect(screen.getByText("Scope Catalogue")).toBeDefined();
    expect(screen.getByText("Register New Scope")).toBeDefined();

    await waitFor(() => {
      expect(screen.getByText("billing:read")).toBeDefined();
      expect(screen.getByText("auth:admin")).toBeDefined();
    });
  });

  it("renders KeyRotateDialog with grace period options and rotation flow", async () => {
    render(
      <KeyRotateDialog
        keyId="key-123"
        keyName="Payment Worker"
        isOpen={true}
        onOpenChange={() => {}}
      />
    );

    expect(screen.getByText(/Rotate API Key/i)).toBeDefined();
    expect(screen.getByText("Zero-Downtime Overlap")).toBeDefined();
    expect(screen.getByText("Confirm Rotation")).toBeDefined();
  });

  it("renders MacaroonInspector and handles sample loading and decoding", async () => {
    render(<MacaroonInspector />);

    expect(screen.getByText("Macaroon & Key Inspector")).toBeDefined();
    const loadSampleBtn = screen.getByText("Load Sample Macaroon");
    expect(loadSampleBtn).toBeDefined();

    fireEvent.click(loadSampleBtn);

    const decodeBtn = screen.getByText("Decode");
    fireEvent.click(decodeBtn);

    await waitFor(() => {
      expect(screen.getByText("Decoded Structure")).toBeDefined();
      expect(screen.getByText("Live Verification Context")).toBeDefined();
    });
  });
});
