import { describe, it, expect, vi, beforeEach } from "vitest";
import React from "react";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

import { CheckSimulator } from "./check-simulator";
import { RelationshipGraph } from "./relationship-graph";
import { CaveatRegistry } from "./caveat-registry";
import { TupleGrid } from "./tuple-grid";

function renderWithClient(ui: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      {ui}
    </QueryClientProvider>
  );
}

describe("Nexus Depth UI Components", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    global.fetch = vi.fn();
  });

  describe("CheckSimulator with DecisionNode Tree", () => {
    it("renders structured decision tree with caveats and rewrite rules", async () => {
      const handleCheck = vi.fn().mockResolvedValue({
        allowed: true,
        reason: "resolved via union rewrite",
        trace: {
          node_id: "root",
          namespace: "document",
          object: "doc_1",
          relation: "view",
          allowed: true,
          rewrite_type: "union",
          caveat: {
            caveat_name: "business_hours",
            allowed: true,
          },
          children: [
            {
              node_id: "child_1",
              namespace: "document",
              object: "doc_1",
              relation: "viewer",
              allowed: true,
              rewrite_type: "this",
            },
          ],
        },
      });

      render(<CheckSimulator onCheck={handleCheck} />);

      fireEvent.change(screen.getByLabelText(/Subject/i), { target: { value: "user:alice" } });
      fireEvent.change(screen.getByLabelText(/Relation/i), { target: { value: "view" } });
      fireEvent.change(screen.getByLabelText(/Object/i), { target: { value: "document:doc_1" } });

      fireEvent.click(screen.getByRole("button", { name: /Check Access/i }));

      await waitFor(() => {
        expect(screen.getByText("ALLOWED")).toBeDefined();
        expect(screen.getByText("document:doc_1#view")).toBeDefined();
        expect(screen.getByText("document:doc_1#viewer")).toBeDefined();
        expect(screen.getByText(/business_hours/i)).toBeDefined();
      });
    });
  });

  describe("RelationshipGraph", () => {
    it("renders neighborhood graph and supports node navigation", async () => {
      render(<RelationshipGraph initialObjectId="document:doc_123" />);

      expect(screen.getByText(/Relationship Graph Explorer/i)).toBeDefined();
      expect(screen.getByText("document:doc_123")).toBeDefined();
      expect(screen.getByText("user:alice")).toBeDefined();

      // Click on target node to navigate
      fireEvent.click(screen.getByText("user:alice"));

      await waitFor(() => {
        expect(screen.getByText("user:alice")).toBeDefined();
        expect(screen.getByText(/Leaf node — no outgoing relations configured/i)).toBeDefined();
      });
    });
  });

  describe("CaveatRegistry", () => {
    it("renders caveats list and define caveat dialog", async () => {
      vi.mocked(global.fetch).mockResolvedValueOnce(
        new Response(
          JSON.stringify([
            {
              name: "is_business_hours",
              cel_expression: "request.time.hour >= 9 && request.time.hour < 18",
              created_at: "2026-09-02T10:00:00Z",
            },
          ]),
          { status: 200, headers: { "Content-Type": "application/json" } }
        )
      );

      renderWithClient(<CaveatRegistry />);

      await waitFor(() => {
        expect(screen.getByText("is_business_hours")).toBeDefined();
        expect(screen.getByText(/request.time.hour >= 9/i)).toBeDefined();
      });

      // Open Define Caveat dialog
      const defineBtn = screen.getByRole("button", { name: /Define Caveat/i });
      fireEvent.click(defineBtn);

      expect(screen.getByLabelText(/Caveat Name/i)).toBeDefined();
      expect(screen.getByLabelText(/CEL Expression/i)).toBeDefined();
    });
  });

  describe("TupleGrid Bulk Import", () => {
    it("opens bulk import dialog and switches format", async () => {
      const handleBulkAdd = vi.fn().mockResolvedValue(undefined);
      render(
        <TupleGrid
          tuples={[
            {
              id: "t1",
              objectType: "document",
              objectId: "doc_1",
              relation: "viewer",
              subjectType: "user",
              subjectId: "alice",
            },
          ]}
          onDelete={vi.fn()}
          onAdd={vi.fn()}
          onBulkAdd={handleBulkAdd}
        />
      );

      const importBtn = screen.getByRole("button", { name: /Bulk Import/i });
      fireEvent.click(importBtn);

      expect(screen.getByText("Bulk Import Relation Tuples")).toBeDefined();
      expect(screen.getByRole("button", { name: /Validate & Commit/i })).toBeDefined();

      // Enter CSV content and commit
      const textarea = screen.getByPlaceholderText(/document,doc_1,viewer,user,alice/i);
      fireEvent.change(textarea, {
        target: { value: "document,doc_2,editor,user,bob" },
      });

      fireEvent.click(screen.getByRole("button", { name: /Validate & Commit/i }));

      await waitFor(() => {
        expect(handleBulkAdd).toHaveBeenCalledWith([
          {
            objectType: "document",
            objectId: "doc_2",
            relation: "editor",
            subjectType: "user",
            subjectId: "bob",
            subjectRelation: "",
          },
        ]);
      });
    });
  });
});
