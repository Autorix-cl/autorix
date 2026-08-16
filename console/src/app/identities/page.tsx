"use client";

import { useState } from "react";
import { Users, Plus, Check, Key, ShieldCheck, UserCheck } from "lucide-react";

interface IdentityItem {
  id: string;
  email: string;
  name: string;
  state: string;
  createdAt: string;
}

export default function IdentitiesPage() {
  const [identities, setIdentities] = useState<IdentityItem[]>([
    {
      id: "e47ac10b-58cc-4372-a567-0e02b2c3d479",
      email: "elena.rostova@autorix.io",
      name: "Elena Rostova",
      state: "active",
      createdAt: "2026-08-16 12:00",
    },
    {
      id: "f81d4fae-7dec-11d0-a765-00a0c91e6bf6",
      email: "alan.turing@autorix.io",
      name: "Alan Turing",
      state: "active",
      createdAt: "2026-08-16 13:30",
    }
  ]);

  const [email, setEmail] = useState("");
  const [firstName, setFirstName] = useState("");
  const [lastName, setLastName] = useState("");
  const [password, setPassword] = useState("");
  const [statusMsg, setStatusMsg] = useState("");

  const handleCreate = (e: React.FormEvent) => {
    e.preventDefault();
    if (!email || !password) return;

    const newId: IdentityItem = {
      id: crypto.randomUUID(),
      email: email,
      name: `${firstName} ${lastName}`.trim() || email.split("@")[0],
      state: "active",
      createdAt: "Just now",
    };

    setIdentities([newId, ...identities]);
    setStatusMsg(`Identity ${email} registered with Argon2id password hashing!`);
    setEmail("");
    setFirstName("");
    setLastName("");
    setPassword("");
    setTimeout(() => setStatusMsg(""), 4000);
  };

  return (
    <>
      <header className="page-header">
        <div>
          <h1 style={{ fontSize: "20px", fontWeight: "700", letterSpacing: "-0.02em" }}>Autorix Ego: Identity Studio</h1>
          <p style={{ fontSize: "13px", color: "var(--text-secondary)", marginTop: "2px" }}>
            User management, JSON Schema traits validation, and Argon2id credentials (:4433).
          </p>
        </div>
        <span className="badge badge-blue">
          <UserCheck size={12} /> REST Headless Active
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
          {/* Create User Form */}
          <div className="card">
            <h2 className="card-title" style={{ marginBottom: "16px", display: "flex", alignItems: "center", gap: "8px" }}>
              <Plus size={16} color="var(--accent-blue)" /> Register New Identity
            </h2>
            
            <form onSubmit={handleCreate}>
              <div className="form-group">
                <label className="form-label">Email Address (Identifier)</label>
                <input 
                  type="email" 
                  className="form-input" 
                  placeholder="user@example.com" 
                  value={email} 
                  onChange={(e) => setEmail(e.target.value)} 
                  required 
                />
              </div>

              <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: "12px" }}>
                <div className="form-group">
                  <label className="form-label">First Name (Trait)</label>
                  <input 
                    type="text" 
                    className="form-input" 
                    placeholder="Ada" 
                    value={firstName} 
                    onChange={(e) => setFirstName(e.target.value)} 
                  />
                </div>
                <div className="form-group">
                  <label className="form-label">Last Name (Trait)</label>
                  <input 
                    type="text" 
                    className="form-input" 
                    placeholder="Lovelace" 
                    value={lastName} 
                    onChange={(e) => setLastName(e.target.value)} 
                  />
                </div>
              </div>

              <div className="form-group">
                <label className="form-label">Password (Argon2id Encoded)</label>
                <input 
                  type="password" 
                  className="form-input" 
                  placeholder="••••••••••••" 
                  value={password} 
                  onChange={(e) => setPassword(e.target.value)} 
                  required 
                />
              </div>

              <button type="submit" className="btn btn-primary" style={{ width: "100%", marginTop: "8px" }}>
                <Key size={14} /> Provision User & Hash Password
              </button>
            </form>
          </div>

          {/* Identity Schema Preview */}
          <div className="card">
            <h2 className="card-title" style={{ marginBottom: "8px", display: "flex", alignItems: "center", gap: "8px" }}>
              <ShieldCheck size={16} color="var(--accent-purple)" /> Dynamic Schema (JSON Schema)
            </h2>
            <p style={{ fontSize: "12px", color: "var(--text-muted)", marginBottom: "16px" }}>
              Profiles are validated against <code style={{ color: "var(--accent-blue)" }}>default.identity.schema.json</code> without database schema changes.
            </p>

            <div className="code-box" style={{ maxHeight: "240px" }}>
{`{
  "$id": "https://schemas.autorix.io/default.identity.schema.json",
  "title": "User Identity",
  "properties": {
    "traits": {
      "properties": {
        "email": { "type": "string", "format": "email" },
        "name": {
          "properties": {
            "first": { "type": "string" },
            "last": { "type": "string" }
          }
        }
      },
      "required": ["email"]
    }
  }
}`}
            </div>
          </div>
        </div>

        {/* Identities Table */}
        <div className="card">
          <div className="card-header">
            <h2 className="card-title">Registered Identities ({identities.length})</h2>
            <span style={{ fontSize: "12px", color: "var(--text-muted)" }}>Storage: autorix_ego (PostgreSQL JSONB)</span>
          </div>

          <div className="table-container">
            <table className="table">
              <thead>
                <tr>
                  <th>Identity UUID</th>
                  <th>Primary Email</th>
                  <th>Display Name</th>
                  <th>State</th>
                  <th>Registered</th>
                </tr>
              </thead>
              <tbody>
                {identities.map((item) => (
                  <tr key={item.id}>
                    <td style={{ fontFamily: "var(--font-mono)", fontSize: "12px", color: "var(--accent-blue)" }}>{item.id}</td>
                    <td style={{ fontWeight: "500" }}>{item.email}</td>
                    <td>{item.name}</td>
                    <td><span className="badge badge-green">{item.state}</span></td>
                    <td style={{ color: "var(--text-muted)", fontSize: "12px" }}>{item.createdAt}</td>
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
