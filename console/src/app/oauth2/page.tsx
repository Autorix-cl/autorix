"use client";

import { useState } from "react";
import { KeyRound, Plus, Check, Shield, Key } from "lucide-react";

interface ClientApp {
  id: string;
  name: string;
  grantTypes: string[];
  scopes: string[];
  isPublic: boolean;
  createdAt: string;
}

export default function OAuth2Page() {
  const [clients, setClients] = useState<ClientApp[]>([
    {
      id: "billing-service",
      name: "Billing Microservice",
      grantTypes: ["client_credentials"],
      scopes: ["invoices:read", "invoices:write"],
      isPublic: false,
      createdAt: "2026-08-16 14:00",
    },
    {
      id: "autorix-spa-dashboard",
      name: "React Web Dashboard",
      grantTypes: ["authorization_code", "refresh_token"],
      scopes: ["openid", "profile", "email"],
      isPublic: true,
      createdAt: "2026-08-16 15:10",
    },
  ]);

  const [clientId, setClientId] = useState("");
  const [clientName, setClientName] = useState("");
  const [clientSecret, setClientSecret] = useState("");
  const [scopes, setScopes] = useState("openid profile email");
  const [isPublic, setIsPublic] = useState(false);
  const [statusMsg, setStatusMsg] = useState("");

  const handleCreate = (e: React.FormEvent) => {
    e.preventDefault();
    if (!clientId || !clientName) return;

    const newClient: ClientApp = {
      id: clientId,
      name: clientName,
      grantTypes: isPublic ? ["authorization_code"] : ["client_credentials"],
      scopes: scopes.split(" "),
      isPublic: isPublic,
      createdAt: "Just now",
    };

    setClients([newClient, ...clients]);
    setStatusMsg(`OAuth2 Client '${clientId}' registered successfully in Janus!`);
    setClientId("");
    setClientName("");
    setClientSecret("");
    setTimeout(() => setStatusMsg(""), 4000);
  };

  return (
    <>
      <header className="page-header">
        <div>
          <h1 style={{ fontSize: "20px", fontWeight: "700", letterSpacing: "-0.02em" }}>Autorix Janus: OAuth2 & JWKS Studio</h1>
          <p style={{ fontSize: "13px", color: "var(--text-secondary)", marginTop: "2px" }}>
            Headless OpenID Connect Provider, PKCE S256 enforcement, and RSA Asymmetric Key management (:4444).
          </p>
        </div>
        <span className="badge badge-amber">
          <Key size={12} /> RS256 JWKS Active
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
          {/* Register Client Form */}
          <div className="card">
            <h2 className="card-title" style={{ marginBottom: "16px", display: "flex", alignItems: "center", gap: "8px" }}>
              <Plus size={16} color="var(--accent-amber)" /> Register OAuth2 Application
            </h2>

            <form onSubmit={handleCreate}>
              <div className="form-group">
                <label className="form-label">Client ID (Unique Slug)</label>
                <input 
                  type="text" 
                  className="form-input" 
                  placeholder="e.g. mobile-app-ios" 
                  value={clientId} 
                  onChange={(e) => setClientId(e.target.value)} 
                  required 
                />
              </div>

              <div className="form-group">
                <label className="form-label">Application Name</label>
                <input 
                  type="text" 
                  className="form-input" 
                  placeholder="e.g. Mobile iOS Client" 
                  value={clientName} 
                  onChange={(e) => setClientName(e.target.value)} 
                  required 
                />
              </div>

              <div className="form-group">
                <label className="form-label">Client Type</label>
                <select 
                  className="form-select" 
                  value={isPublic ? "public" : "confidential"} 
                  onChange={(e) => setIsPublic(e.target.value === "public")}
                >
                  <option value="confidential">Confidential (Backend M2M - Requires Secret)</option>
                  <option value="public">Public (SPA / Mobile - Requires PKCE S256)</option>
                </select>
              </div>

              {!isPublic && (
                <div className="form-group">
                  <label className="form-label">Client Secret</label>
                  <input 
                    type="password" 
                    className="form-input" 
                    placeholder="SuperSecretPassword123!" 
                    value={clientSecret} 
                    onChange={(e) => setClientSecret(e.target.value)} 
                  />
                </div>
              )}

              <div className="form-group">
                <label className="form-label">Allowed Scopes</label>
                <input 
                  type="text" 
                  className="form-input" 
                  value={scopes} 
                  onChange={(e) => setScopes(e.target.value)} 
                />
              </div>

              <button 
                type="submit" 
                className="btn btn-primary" 
                style={{ width: "100%", background: "var(--accent-amber)", borderColor: "var(--accent-amber)" }}
              >
                <KeyRound size={14} /> Register Application
              </button>
            </form>
          </div>

          {/* Active JWKS Keys Viewer */}
          <div className="card">
            <h2 className="card-title" style={{ marginBottom: "8px", display: "flex", alignItems: "center", gap: "8px" }}>
              <Shield size={16} color="var(--accent-green)" /> Active JWKS Public Key (`/.well-known/jwks.json`)
            </h2>
            <p style={{ fontSize: "12px", color: "var(--text-muted)", marginBottom: "16px" }}>
              Microservices and Aegis use this RSA key to verify JWT signatures in memory without network calls.
            </p>

            <div className="code-box" style={{ maxHeight: "280px" }}>
{`{
  "keys": [
    {
      "kty": "RSA",
      "use": "sig",
      "alg": "RS256",
      "kid": "autorix-master-key-2026",
      "n": "u1P5tQ98df23...xJ923kLm8",
      "e": "AQAB"
    }
  ]
}`}
            </div>
          </div>
        </div>

        {/* Clients Table */}
        <div className="card">
          <div className="card-header">
            <h2 className="card-title">Registered OAuth2 Clients ({clients.length})</h2>
            <span style={{ fontSize: "12px", color: "var(--text-muted)" }}>Storage: autorix_janus</span>
          </div>

          <div className="table-container">
            <table className="table">
              <thead>
                <tr>
                  <th>Client ID</th>
                  <th>Name</th>
                  <th>Grant Types</th>
                  <th>Scopes</th>
                  <th>Type</th>
                  <th>Created</th>
                </tr>
              </thead>
              <tbody>
                {clients.map((c) => (
                  <tr key={c.id}>
                    <td style={{ fontFamily: "var(--font-mono)", fontSize: "12px", color: "var(--accent-amber)" }}>{c.id}</td>
                    <td style={{ fontWeight: "500" }}>{c.name}</td>
                    <td>
                      {c.grantTypes.map((g) => (
                        <span key={g} className="badge badge-amber" style={{ fontSize: "10px", marginRight: "4px" }}>{g}</span>
                      ))}
                    </td>
                    <td style={{ fontFamily: "var(--font-mono)", fontSize: "11px", color: "var(--text-muted)" }}>
                      {c.scopes.join(" ")}
                    </td>
                    <td>
                      <span className={c.isPublic ? "badge badge-blue" : "badge badge-purple"}>
                        {c.isPublic ? "Public (PKCE)" : "Confidential"}
                      </span>
                    </td>
                    <td style={{ color: "var(--text-muted)", fontSize: "12px" }}>{c.createdAt}</td>
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
