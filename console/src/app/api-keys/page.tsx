"use client";

import { useState } from "react";
import { Layers, Plus, Check, Shield, Lock, RefreshCw } from "lucide-react";

interface KeyItem {
  id: string;
  name: string;
  prefix: string;
  keyMasked: string;
  ownerId: string;
  caveatsCount: number;
  createdAt: string;
}

export default function ApiKeysPage() {
  const [keys, setKeys] = useState<KeyItem[]>([
    {
      id: "vulcan_k1",
      name: "Stripe Webhook Worker",
      prefix: "av_live_",
      keyMasked: "av_live_9a8b...1f2e",
      ownerId: "service-worker-01",
      caveatsCount: 2,
      createdAt: "2026-08-16 10:00",
    },
    {
      id: "vulcan_k2",
      name: "Staging Testing Key",
      prefix: "av_test_",
      keyMasked: "av_test_3c4d...8a9b",
      ownerId: "qa-team",
      caveatsCount: 0,
      createdAt: "2026-08-16 11:20",
    }
  ]);

  const [keyName, setKeyName] = useState("");
  const [ownerId, setOwnerId] = useState("");
  const [isLive, setIsLive] = useState(true);
  const [newlyCreatedKey, setNewlyCreatedKey] = useState<string | null>(null);

  // Attenuation state
  const [selectedKeyForAttenuation, setSelectedKeyForAttenuation] = useState("av_live_9a8b...1f2e");
  const [newCaveat, setNewCaveat] = useState("ip = 192.168.1.50");
  const [attenuatedToken, setAttenuatedToken] = useState<string | null>(null);

  const handleCreate = (e: React.FormEvent) => {
    e.preventDefault();
    if (!keyName) return;

    const randomSuffix = Array.from(crypto.getRandomValues(new Uint8Array(16)))
      .map((b) => b.toString(16).padStart(2, "0"))
      .join("");
    const generatedRawKey = `${isLive ? "av_live_" : "av_test_"}${randomSuffix}`;

    const newKey: KeyItem = {
      id: `vulcan_${crypto.randomUUID().slice(0, 8)}`,
      name: keyName,
      prefix: isLive ? "av_live_" : "av_test_",
      keyMasked: `${generatedRawKey.slice(0, 12)}...${generatedRawKey.slice(-4)}`,
      ownerId: ownerId || "system",
      caveatsCount: 0,
      createdAt: "Just now",
    };

    setKeys([newKey, ...keys]);
    setNewlyCreatedKey(generatedRawKey);
    setKeyName("");
    setOwnerId("");
  };

  const handleAttenuate = () => {
    if (!newCaveat) return;
    const signature = Array.from(crypto.getRandomValues(new Uint8Array(8)))
      .map((b) => b.toString(16).padStart(2, "0"))
      .join("");
    setAttenuatedToken(`macaroon_v1:${selectedKeyForAttenuation}:[${newCaveat}]:hmac_${signature}`);
  };

  return (
    <>
      <header className="page-header">
        <div>
          <h1 style={{ fontSize: "20px", fontWeight: "700", letterSpacing: "-0.02em" }}>Autorix Vulcan: API Keys & Macaroons</h1>
          <p style={{ fontSize: "13px", color: "var(--text-secondary)", marginTop: "2px" }}>
            Scannable prefixed keys and decentralized capability tokens with chained HMAC attenuation (:4466).
          </p>
        </div>
        <span className="badge badge-blue">
          <Layers size={12} /> Macaroon HMAC Active
        </span>
      </header>

      <div className="page-body">
        {newlyCreatedKey && (
          <div style={{ padding: "16px", backgroundColor: "rgba(16, 185, 129, 0.15)", border: "1px solid var(--accent-green)", borderRadius: "var(--radius-md)", marginBottom: "24px" }}>
            <div style={{ display: "flex", alignItems: "center", gap: "8px", color: "var(--accent-green)", fontWeight: "600", marginBottom: "8px" }}>
              <Check size={18} /> API Key Generated (Copy now, it will not be shown again):
            </div>
            <div className="code-box" style={{ color: "#fff", background: "#000", fontSize: "14px" }}>
              {newlyCreatedKey}
            </div>
          </div>
        )}

        <div className="grid-2">
          {/* Create Key */}
          <div className="card">
            <h2 className="card-title" style={{ marginBottom: "16px", display: "flex", alignItems: "center", gap: "8px" }}>
              <Plus size={16} color="var(--accent-cyan)" /> Generate API Key
            </h2>

            <form onSubmit={handleCreate}>
              <div className="form-group">
                <label className="form-label">Key Name / Description</label>
                <input 
                  type="text" 
                  className="form-input" 
                  placeholder="e.g. Production Ingestion Service" 
                  value={keyName} 
                  onChange={(e) => setKeyName(e.target.value)} 
                  required 
                />
              </div>

              <div className="form-group">
                <label className="form-label">Owner / Service Subject</label>
                <input 
                  type="text" 
                  className="form-input" 
                  placeholder="e.g. billing-service" 
                  value={ownerId} 
                  onChange={(e) => setOwnerId(e.target.value)} 
                />
              </div>

              <div className="form-group">
                <label className="form-label">Environment Prefix</label>
                <select 
                  className="form-select" 
                  value={isLive ? "live" : "test"} 
                  onChange={(e) => setIsLive(e.target.value === "live")}
                >
                  <option value="live">Live (`av_live_...` Production)</option>
                  <option value="test">Test (`av_test_...` Sandbox)</option>
                </select>
              </div>

              <button 
                type="submit" 
                className="btn btn-primary" 
                style={{ width: "100%", background: "var(--accent-cyan)", borderColor: "var(--accent-cyan)" }}
              >
                <Lock size={14} /> Generate Scannable API Key
              </button>
            </form>
          </div>

          {/* Macaroon Attenuation Studio */}
          <div className="card">
            <h2 className="card-title" style={{ marginBottom: "8px", display: "flex", alignItems: "center", gap: "8px" }}>
              <RefreshCw size={16} color="var(--accent-purple)" /> Macaroon Attenuation Studio
            </h2>
            <p style={{ fontSize: "12px", color: "var(--text-muted)", marginBottom: "16px" }}>
              Attenuate permissions locally by adding caveats without round-tripping to the database.
            </p>

            <div className="form-group">
              <label className="form-label">Select Base Key</label>
              <select 
                className="form-select" 
                value={selectedKeyForAttenuation} 
                onChange={(e) => setSelectedKeyForAttenuation(e.target.value)}
              >
                {keys.map((k) => (
                  <option key={k.id} value={k.keyMasked}>{k.name} ({k.keyMasked})</option>
                ))}
              </select>
            </div>

            <div className="form-group">
              <label className="form-label">Add First-Party Caveat</label>
              <select 
                className="form-select" 
                value={newCaveat} 
                onChange={(e) => setNewCaveat(e.target.value)}
              >
                <option value="time_before = 2026-08-17T00:00:00Z">Time Expiry (time_before = 2026-08-17T00:00:00Z)</option>
                <option value="ip = 192.168.1.50">IP Whitelist (ip = 192.168.1.50)</option>
                <option value="method = GET">Read-Only (method = GET)</option>
                <option value="path_prefix = /api/v1/public">Scope Limiter (path_prefix = /api/v1/public)</option>
              </select>
            </div>

            <button 
              type="button" 
              className="btn btn-secondary" 
              style={{ width: "100%", marginBottom: "12px" }}
              onClick={handleAttenuate}
            >
              Compute HMAC Chain Signature
            </button>

            {attenuatedToken && (
              <div className="code-box" style={{ fontSize: "11px", color: "var(--accent-green)" }}>
                {attenuatedToken}
              </div>
            )}
          </div>
        </div>

        {/* Keys Table */}
        <div className="card">
          <div className="card-header">
            <h2 className="card-title">Issued API Keys ({keys.length})</h2>
            <span style={{ fontSize: "12px", color: "var(--text-muted)" }}>Storage: autorix_vulcan (SHA-256 Hashes)</span>
          </div>

          <div className="table-container">
            <table className="table">
              <thead>
                <tr>
                  <th>Key Name</th>
                  <th>Masked Token</th>
                  <th>Owner Subject</th>
                  <th>Environment</th>
                  <th>Created</th>
                </tr>
              </thead>
              <tbody>
                {keys.map((k) => (
                  <tr key={k.id}>
                    <td style={{ fontWeight: "500" }}>{k.name}</td>
                    <td style={{ fontFamily: "var(--font-mono)", fontSize: "12px", color: "var(--accent-cyan)" }}>{k.keyMasked}</td>
                    <td>{k.ownerId}</td>
                    <td>
                      <span className={k.prefix === "av_live_" ? "badge badge-green" : "badge badge-amber"}>
                        {k.prefix === "av_live_" ? "Production" : "Sandbox"}
                      </span>
                    </td>
                    <td style={{ color: "var(--text-muted)", fontSize: "12px" }}>{k.createdAt}</td>
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
