import { render, screen, waitFor } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import NexusPage from "./page";

// Mock the context
vi.mock("@/lib/capabilities/capability-context", () => ({
  useCapabilities: () => ({
    isEngineConnected: (engine: string) => engine === "nexus",
  }),
}));

// Mock translation
vi.mock("@/lib/i18n", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}));

// Mock React Query hooks
vi.mock("@/hooks/use-nexus", () => ({
  useNexusSchema: vi.fn().mockReturnValue({ data: "definition user {}", isLoading: false }),
  useSaveNexusSchema: vi.fn().mockReturnValue({ mutateAsync: vi.fn() }),
  useNexusTuples: vi.fn().mockReturnValue({ data: [], isLoading: false }),
  useAddNexusTuple: vi.fn().mockReturnValue({ mutateAsync: vi.fn() }),
  useDeleteNexusTuples: vi.fn().mockReturnValue({ mutateAsync: vi.fn() }),
  useNexusCheck: vi.fn().mockReturnValue({ mutateAsync: vi.fn() }),
}));

describe("NexusPage", () => {
  it("renders the schema editor and relation tuples", async () => {
    render(<NexusPage />);
    expect(screen.getByText("Nexus Playground")).toBeInTheDocument();
    expect(screen.getByText("Authorization Schema")).toBeInTheDocument();
    await waitFor(() => {
      expect(screen.getByText("Relation Tuples")).toBeInTheDocument();
      expect(screen.getByText("Check Simulator (Playground)")).toBeInTheDocument();
    });
  });
});
