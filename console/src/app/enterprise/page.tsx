"use client";

import { useState } from "react";
import { Building2, Plus, Check, Download, ShieldCheck, RefreshCw } from "lucide-react";

interface SAMLProviderItem {
  id: string;
  name: string;
  idpEntityId: string;
  ssoUrl: string;
  enabled: boolean;
  createdAt: string;
}

interface SCIMUserItem {
  id: string;
  externalId: string;
  userName: string;
  email: string;
  active: boolean;
}

export default function EnterprisePage() {
  const [providers, setProviders] = useState<SAMLProviderItem[]>([
    {
      id: "okta-corporate",
      name: "Okta Enterprise SSO",
      idpEntityId: "http://www.okta.com/exk998877",
      ssoUrl: "https://company.okta.com/app/sso/saml",
      enabled: true,
      createdAt: "2026-08-16 09:00",
    }
  ]);

  const [scimUsers] = useState<SCIMUserItem[]>([
    {
      id: "e47ac10b-58cc-4372-a567-0e02b2c3d479",
      externalId: "okta_998877",
      userName: "margaret.hamilton@nasa.gov",
      email: "margaret.hamilton@nasa.gov",
      active: true,
    }
  ]);

  const [providerId, setProviderId] = useState("");
  const [providerName, setProviderName] = useState("");
  const [idpEntityId, setIdpEntityId] = useState("");
  const [idpSsoUrl, setIdpSsoUrl] = useState("");
  const [statusMsg, setStatusMsg] = useState("");

  const handleCreateProvider = (e: React.FormEvent) => {
    e.preventDefault();
    if (!providerId || !idpSsoUrl) return;

    const newP: SAMLProviderItem = {
      id: providerId,
      name: providerName || providerId,
      idpEntityId: idpEntityId || `https://idp.example.com/${providerId}`,
      ssoUrl: idpSsoUrl,
      enabled: true,
      createdAt: "Just now",
    };

    setProviders([newP, ...providers]);
    setStatusMsg(`SAML Provider '${providerId}' registered in Hermes!`);
    setProviderId("");
    setProviderName("");
    setIdpEntityId("");
    setIdpSsoUrl("");
    setTimeout(() => setStatusMsg(""), 4000);
  };

  return (
    <>
      <header className="page-header">
        <div>
          <h1 style={{ fontSize: "20px", fontWeight: "700", letterSpacing: "-0.02em" }}>Autorix Hermes: SAML 2.0 & SCIM 2.0 Studio</h1>
          <p style={{ fontSize: "13px", color: "var(--text-secondary)", marginTop: "2px" }}>
            Enterprise SSO Federation (Okta, Azure AD) and Directory Provisioning (RFC 7643 / RFC 7644) (:4477).
          </p>
        </div>
        <span className="badge badge-amber">
          <Building2 size={12} /> SAML SP Active
        </span>
      </header>

      <div className="page-body">
        {statusMsg && (
          <div style={{ padding: "12px 16px", backgroundColor: "rgba(16, 185, 129, 0.15)", border: "1px solid var(--accent-green)", borderRadius: "var(--radius-md)", color: "var(--accent-green)", marginBottom: "24px", display: "flex", alignItems: "center", gap: "8px", fontSize: "13px" }}>
            <Check size={16} />
            {statusMsg}
          </div>
        )}

        <div className="grid-2">
          {/* Add SAML Provider */}
          <div className="card">
            <h2 className="card-title" style={{ marginBottom: "16px", display: "flex", alignItems: "center", gap: "8px" }}>
              <Plus size={16} color="var(--accent-rose)" /> Register SAML 2.0 Identity Provider
            </h2>

            <form onSubmit={handleCreateProvider}>
              <div className="form-group">
                <label className="form-label">Provider ID (Slug)</label>
                <input 
                  type="text" 
                  className="form-input" 
                  placeholder="e.g. azure-ad-corporate" 
                  value={providerId} 
                  onChange={(e) => setProviderId(e.target.value)} 
                  required 
                />
              </div>

              <div className="form-group">
                <label className="form-label">Display Name</label>
                <input 
                  type="text" 
                  className="form-input" 
                  placeholder="e.g. Microsoft Entra ID" 
                  value={providerName} 
                  onChange={(e) => setProviderName(e.target.value)} 
                />
              </div>

              <div className="form-group">
                <label className="form-label">IdP Entity ID</label>
                <input 
                  type="text" 
                  className="form-input" 
                  placeholder="https://sts.windows.net/tenant-id/" 
                  value={idpEntityId} 
                  onChange={(e) => setIdpEntityId(e.target.value)} 
                />
              </div>

              <div className="form-group">
                <label className="form-label">IdP Single Sign-On URL</label>
                <input 
                  type="url" 
                  className="form-input" 
                  placeholder="https://login.microsoftonline.com/.../saml2" 
                  value={idpSsoUrl} 
                  onChange={(e) => setIdpSsoUrl(e.target.value)} 
                  required 
                />
              </div>

              <button 
                type="submit" 
                className="btn btn-primary" 
                style={{ width: "100%", background: "var(--accent-rose)", borderColor: "var(--accent-rose)" }}
              >
                <ShieldCheck size={14} /> Register IdP & Setup SP
              </button>
            </form>
          </div>

          {/* SP Metadata & SCIM Endpoints */}
          <div className="card">
            <h2 className="card-title" style={{ marginBottom: "8px" }}>Service Provider (SP) Metadata</h2>
            <p style={{ fontSize: "12px", color: "var(--text-muted)", marginBottom: "16px" }}>
              Upload this metadata XML into your Okta or Azure AD admin portal.
            </p>

            <div className="code-box" style={{ maxHeight: "180px", marginBottom: "16px" }}>
{`<?xml version="1.0" encoding="UTF-8"?>
<md:EntityDescriptor xmlns:md="urn:oasis:names:tc:SAML:2.0:metadata" entityID="https://hermes.autorix.io/sp">
  <md:SPSSODescriptor AuthnRequestsSigned="false" WantAssertionsSigned="true">
    <md:AssertionConsumerService 
      Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" 
      Location="http://localhost:4477/saml/acs" index="1" isDefault="true"/>
  </md:SPSSODescriptor>
</md:EntityDescriptor>`}
            </div>

            <div style={{ padding: "12px", background: "var(--bg-tertiary)", borderRadius: "var(--radius-md)" }}>
              <div className="form-label" style={{ color: "var(--accent-blue)", marginBottom: "4px" }}>SCIM 2.0 Base URL</div>
              <div style={{ fontFamily: "var(--font-mono)", fontSize: "12px" }}>http://localhost:4477/scim/v2</div>
            </div>
          </div>
        </div>

        {/* SCIM Synced Users Table */}
        <div className="card">
          <div className="card-header">
            <h2 className="card-title">SCIM 2.0 Synchronized Directory ({scimUsers.length})</h2>
            <span style={{ fontSize: "12px", color: "var(--text-muted)" }}>Auto-provisioned via RFC 7644</span>
          </div>

          <div className="table-container">
            <table className="table">
              <thead>
                <tr>
                  <th>SCIM UUID</th>
                  <th>External IdP ID</th>
                  <th>Username</th>
                  <th>Email</th>
                  <th>Status</th>
                </tr>
              </thead>
              <tbody>
                {scimUsers.map((u) => (
                  <tr key={u.id}>
                    <td style={{ fontFamily: "var(--font-mono)", fontSize: "12px", color: "var(--accent-rose)" }}>{u.id}</td>
                    <td style={{ fontFamily: "var(--font-mono)", fontSize: "12px" }}>{u.externalId}</td>
                    <td style={{ fontWeight: "500" }}>{u.userName}</td>
                    <td>{u.email}</td>
                    <td><span className="badge badge-green">Active</span></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </>
  );
}
