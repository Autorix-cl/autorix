import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import * as React from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { SchemaStudio } from "./schema-studio";

// Mock CodeEditor since CodeMirror needs browser DOM canvas/measurements
vi.mock("@/components/ui/code-editor", () => ({
  CodeEditor: ({ value, onChange }: { value: string; onChange: (v: string) => void }) => (
    <textarea
      data-testid="mock-code-editor"
      value={value}
      onChange={(e) => onChange(e.target.value)}
    />
  ),
}));

describe("SchemaStudio Component", () => {
  let queryClient: QueryClient;

  beforeEach(() => {
    queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    vi.stubGlobal(
      "fetch",
      vi.fn().mockImplementation((url: string) => {
        if (url.includes("/api/identities/schemas")) {
          return Promise.resolve({
            ok: true,
            json: async () => [
              {
                id: "default",
                name: "User Identity",
                schema: {
                  type: "object",
                  properties: {
                    traits: {
                      type: "object",
                      properties: {
                        email: { type: "string", format: "email", title: "Email Address" },
                      },
                      required: ["email"],
                    },
                  },
                },
              },
            ],
          });
        }
        return Promise.resolve({ ok: true, json: async () => ({}) });
      })
    );
  });

  it("renders catalog and loads the default schema", async () => {
    render(
      <QueryClientProvider client={queryClient}>
        <SchemaStudio />
      </QueryClientProvider>
    );

    await waitFor(() => {
      expect(screen.getByText("Ego Identity Schemas Catalog")).toBeInTheDocument();
      expect(screen.getByText("User Identity")).toBeInTheDocument();
    });

    expect(screen.getByText("Interactive Form Preview")).toBeInTheDocument();
    expect(screen.getByText("Valid Schema")).toBeInTheDocument();
  });

  it("allows switching to preset templates and updates the schema editor", async () => {
    render(
      <QueryClientProvider client={queryClient}>
        <SchemaStudio />
      </QueryClientProvider>
    );

    await waitFor(() => {
      expect(screen.getByText("B2B")).toBeInTheDocument();
    });

    const b2bButton = screen.getByText("B2B");
    fireEvent.click(b2bButton);

    const schemaIdInput = screen.getByLabelText(/Schema Identifier/i);
    expect(schemaIdInput).toHaveValue("b2b_partner");

    const editor = screen.getByTestId("mock-code-editor") as HTMLTextAreaElement;
    expect(editor.value).toContain("B2B Partner Identity");
  });

  it("shows error badge when JSON schema syntax is invalid", async () => {
    render(
      <QueryClientProvider client={queryClient}>
        <SchemaStudio />
      </QueryClientProvider>
    );

    await waitFor(() => {
      expect(screen.getByText("User Identity")).toBeInTheDocument();
      expect(screen.getByTestId("mock-code-editor")).toBeInTheDocument();
    });

    const editor = screen.getByTestId("mock-code-editor");
    fireEvent.change(editor, { target: { value: "{ invalid json" } });

    await waitFor(() => {
      expect(screen.getByText("Syntax Error")).toBeInTheDocument();
    });
  });
});
