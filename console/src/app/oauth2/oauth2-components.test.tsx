import { describe, it, expect, vi, beforeEach } from "vitest";
import React from "react";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

import { TokenInspector } from "./token-inspector";
import { ScopeCatalogue } from "./scope-catalogue";
import { KeyManager } from "./key-manager";
import { ClientDetailDialog } from "./client-detail-dialog";

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
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false, networkMode: "always" },
    },
  });

  return render(
    <I18nProvider>
      <QueryClientProvider client={queryClient}>
        {ui}
      </QueryClientProvider>
    </I18nProvider>
  );
}

describe("OAuth2 UI Components", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("TokenInspector", () => {
    it("renders input and performs introspection", async () => {
      vi.mocked(fetchAndParse).mockResolvedValue({
        ok: true,
        data: {
          active: true,
          sub: "user-test-123",
          client_id: "test-client",
          scope: "openid profile",
          exp: 1770000000,
        },
      });

      renderWithClient(<TokenInspector initialToken="eyJhbGciOiJSUzI1NiJ9.test" />);

      expect(screen.getByText(/Token Introspection & Revocation/i)).toBeDefined();

      const introspectBtn = screen.getByRole("button", { name: /Introspect Token/i });
      fireEvent.click(introspectBtn);

      await waitFor(() => {
        expect(screen.getByText(/Introspection Result/i)).toBeDefined();
        expect(screen.getAllByText(/user-test-123/i).length).toBeGreaterThanOrEqual(1);
      });
    });
  });


  describe("ScopeCatalogue", () => {

    it("renders scopes list and new scope dialog", async () => {
      vi.mocked(fetchAndParse).mockResolvedValue({
        ok: true,
        data: [
          { name: "openid", description: "OpenID Connect", claims: ["sub"], created_at: "2026-01-01T00:00:00Z" },
        ],
      });

      renderWithClient(<ScopeCatalogue />);

      await waitFor(() => {
        expect(screen.getByText("openid")).toBeDefined();
        expect(screen.getByText("OpenID Connect")).toBeDefined();
      });

      const newBtn = screen.getByRole("button", { name: /New Scope/i });
      fireEvent.click(newBtn);

      expect(screen.getByText(/Register New Scope/i)).toBeDefined();
      expect(screen.getByLabelText(/Scope Name/i)).toBeDefined();
    });
  });

  describe("KeyManager", () => {
    it("renders active JWKS keys and handles rotation", async () => {
      vi.mocked(fetchAndParse).mockImplementation(async (url: string) => {
        if (url === "/api/oauth2/jwks") {
          return {
            ok: true,
            data: {
              keys: [
                { kid: "key-primary-1", kty: "RSA", use: "sig", alg: "RS256", n: "abc", e: "AQAB" },
              ],
            },
          };
        }
        if (url === "/api/oauth2/keys/rotate") {
          return {
            ok: true,
            data: {
              status: "rotated",
              new_kid: "key-primary-2",
              active_keys_count: 2,
            },
          };
        }
        return { ok: true, data: null };
      });

      renderWithClient(<KeyManager />);


      await waitFor(() => {
        expect(screen.getByText("key-primary-1")).toBeDefined();
      });

      const rotateBtn = screen.getByRole("button", { name: /Rotate Signing Key/i });
      fireEvent.click(rotateBtn);

      await waitFor(() => {
        expect(screen.getByText(/New primary signing key generated/i)).toBeDefined();
      });
    });
  });

  describe("ClientDetailDialog", () => {
    it("renders client details and allows secret rotation", async () => {
      const mockClient = {
        client_id: "client-dashboard",
        client_name: "Dashboard App",
        grant_types: ["authorization_code"],
        response_types: ["code"],
        redirect_uris: ["https://dash.example.com/cb"],
        scopes: ["openid"],
        allowed_audiences: [],
        is_public: false,
        created_at: "2026-01-01T00:00:00Z",
        updated_at: "2026-01-01T00:00:00Z",
      };

      vi.mocked(fetchAndParse).mockResolvedValue({
        ok: true,
        data: {
          client_id: "client-dashboard",
          client_secret: "secret-new-xyz-987",
          previous_secret_expires_at: "2026-09-03T12:00:00Z",
        },
      });

      renderWithClient(
        <ClientDetailDialog
          client={mockClient}
          open={true}
          onOpenChange={() => {}}
        />
      );

      expect(screen.getByText("Dashboard App")).toBeDefined();
      expect(screen.getByText("Zero-Downtime Secret Rotation")).toBeDefined();

      const rotateBtn = screen.getByRole("button", { name: /Rotate Secret/i });
      fireEvent.click(rotateBtn);

      await waitFor(() => {
        expect(screen.getByText(/New Secret Generated — Store it now!/i)).toBeDefined();
        expect(screen.getByText("secret-new-xyz-987")).toBeDefined();
      });
    });
  });
});
