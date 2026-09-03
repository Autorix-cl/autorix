import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import * as React from "react";
import { SAMLProvidersTable } from "./saml-providers-table";
import { SAMLProviderSheet } from "./saml-provider-sheet";
import { SAMLConnectionWizard } from "./saml-connection-wizard";
import { SCIMSyncMonitor } from "./scim-sync-monitor";
import type { SAMLProvider, SCIMUser } from "@/lib/api/schemas/hermes";

describe("Hermes Enterprise UI Components", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    global.fetch = vi.fn();
  });

  const sampleProvider: SAMLProvider = {
    id: "okta-corp",
    display_name: "Okta Corporate",
    idp_entity_id: "https://okta.example.com/entity",
    idp_sso_url: "https://okta.example.com/sso",
    idp_certificate_pem: "cert-pem",
    sp_entity_id: "https://autorix.io/sp",
    attribute_mapping: { email: "mail", name: "displayName" },
    enabled: true,
    certificates: [
      {
        subject: "CN=Okta Corp",
        issuer: "CN=Okta CA",
        serial_number: "999",
        not_before: "2025-01-01T00:00:00Z",
        not_after: "2027-01-01T00:00:00Z",
        expired: false,
        expiring_soon: false,
        days_until_expiry: 150,
      },
    ],
    created_at: "2026-09-02T10:00:00Z",
    updated_at: "2026-09-02T10:00:00Z",
  };

  it("renders SAMLProvidersTable and triggers configure sheet", async () => {
    render(<SAMLProvidersTable providers={[sampleProvider]} />);

    expect(screen.getByText("Okta Corporate")).toBeDefined();
    expect(screen.getByText("Valid")).toBeDefined();

    const configureBtn = screen.getByText("Configure");
    fireEvent.click(configureBtn);

    await waitFor(() => {
      expect(screen.getByText("IdP Signing Certificates")).toBeDefined();
    });
  });

  it("renders SAMLProviderSheet with attribute mapping and certificate details", () => {
    render(
      <SAMLProviderSheet
        provider={sampleProvider}
        isOpen={true}
        onOpenChange={() => {}}
      />
    );

    expect(screen.getByText("SAML Attribute Mapping")).toBeDefined();
    expect(screen.getByText("CN=Okta Corp")).toBeDefined();
    expect(screen.getByText("Save Changes")).toBeDefined();
  });

  it("navigates SAMLConnectionWizard steps", async () => {
    render(<SAMLConnectionWizard isOpen={true} onOpenChange={() => {}} />);

    expect(screen.getByText("SAML Connection Setup Wizard")).toBeDefined();
    expect(screen.getByText("1. Identity Provider")).toBeDefined();
  });

  it("renders SCIMSyncMonitor and switches tabs", async () => {
    const sampleUser: SCIMUser = {
      schemas: ["urn:ietf:params:scim:schemas:core:2.0:User"],
      id: "u-1",
      userName: "john.doe",
      emails: [{ value: "john@corp.com", type: "work", primary: true }],
      active: true,
      meta: {
        resourceType: "User",
        created: "2026-09-02T10:00:00Z",
        lastModified: "2026-09-02T10:00:00Z",
        location: "/scim/v2/Users/u-1",
      },
    };

    render(<SCIMSyncMonitor users={[sampleUser]} />);

    expect(screen.getByText("john.doe")).toBeDefined();
    expect(screen.getByText("john@corp.com")).toBeDefined();

    const groupsBtn = screen.getByRole("button", { name: /Groups/i });
    fireEvent.click(groupsBtn);

    await waitFor(() => {
      expect(screen.getByText("Add Group")).toBeDefined();
    });
  });
});
