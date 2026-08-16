"use client";

import { useState } from "react";
import { Shield, Play, ArrowRight, CheckCircle2, Lock } from "lucide-react";

interface RuleItem {
  id: string;
  match: {
    methods: string[];
    url: string;
  };
  authenticator: string;
  authorizer: string;
  mutator: string;
  upstream: string;
}

export default function ProxyRulesPage() {
  const [rules] = useState<RuleItem[]>([
    {
      id: "rule-documents-api",
      match: {
        methods: ["GET", "POST"],
        url: "/api/v1/documents/<.*>",
      },
      authenticator: "jwt (Janus JWKS)",
      authorizer: "nexus_rebac (document:<id>#viewer)",
      mutator: "header (inject X-User-ID)",
      upstream: "http://documents-backend:8080",
    },
    {
      id: "rule-public-health",
      match: {
        methods: ["GET"],
        url: "/health",
      },
      authenticator: "anonymous (allow all)",
      authorizer: "allow (pass-through)",
      mutator: "noop",
      upstream: "http://core-backend:8080",
    },
  ]);

  const [testPath, setTestPath] = useState("/api/v1/documents/financial_report_2026");
  const [testMethod, setTestMethod] = useState("GET");
  const [matchedRule, setMatchedRule] = useState<RuleItem | null>(rules[0]);

  const handleTestMatch = () => {
    if (testPath.startsWith("/api/v1/documents/")) {
      setMatchedRule(rules[0]);
    } else if (testPath === "/health") {
      setMatchedRule(rules[1]);
    } else {
      setMatchedRule(null);
    }
  };

  return (
    <>
      <header className="page-header">
        <div>
          <h1 style={{ fontSize: "20px", fontWeight: "700", letterSpacing: "-0.02em" }}>Autorix Aegis: Zero Trust Proxy Rules</h1>
          <p style={{ fontSize: "13px", color: "var(--text-secondary)", marginTop: "2px" }}>
            Declarative request interception, pipeline routing (Auth $\rightarrow$ Authz $\rightarrow$ Mutate) (:4455).
          </p>
        </div>
        <span className="badge badge-green">
          <Shield size={12} /> PEP Reverse Proxy Active
        </span>
      </header>

      <div className="page-body">
        {/* Rule Tester Simulator */}
        <div className="card">
          <h2 className="card-title" style={{ marginBottom: "16px", display: "flex", alignItems: "center", gap: "8px" }}>
            <Play size={16} color="var(--accent-green)" /> Proxy Routing Simulator
          </h2>

          <div style={{ display: "grid", gridTemplateColumns: "140px 1fr 140px", gap: "12px", marginBottom: "20px" }}>
            <div>
              <label className="form-label">HTTP Method</label>
              <select 
                className="form-select" 
                value={testMethod} 
                onChange={(e) => setTestMethod(e.target.value)}
              >
                <option value="GET">GET</option>
                <option value="POST">POST</option>
                <option value="PUT">PUT</option>
                <option value="DELETE">DELETE</option>
              </select>
            </div>

            <div>
              <label className="form-label">Incoming Request Path</label>
              <input 
                type="text" 
                className="form-input" 
                value={testPath} 
                onChange={(e) => setTestPath(e.target.value)} 
              />
            </div>

            <div style={{ display: "flex", alignItems: "flex-end" }}>
              <button 
                type="button" 
                className="btn btn-primary" 
                style={{ width: "100%", background: "var(--accent-green)", borderColor: "var(--accent-green)" }}
                onClick={handleTestMatch}
              >
                Simulate Match
              </button>
            </div>
          </div>

          {matchedRule ? (
            <div style={{ padding: "16px", background: "var(--bg-primary)", border: "1px solid var(--border-subtle)", borderRadius: "var(--radius-md)" }}>
              <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: "12px" }}>
                <span className="badge badge-green">Rule Matched: {matchedRule.id}</span>
                <span style={{ fontSize: "12px", color: "var(--text-muted)", fontFamily: "var(--font-mono)" }}>Upstream: {matchedRule.upstream}</span>
              </div>

              <div style={{ display: "grid", gridTemplateColumns: "repeat(3, 1fr)", gap: "16px", marginTop: "12px" }}>
                <div style={{ padding: "12px", background: "var(--bg-tertiary)", borderRadius: "var(--radius-md)" }}>
                  <div className="form-label" style={{ color: "var(--accent-blue)" }}>1. Authenticator</div>
                  <div style={{ fontSize: "13px", fontWeight: "600" }}>{matchedRule.authenticator}</div>
                </div>

                <div style={{ padding: "12px", background: "var(--bg-tertiary)", borderRadius: "var(--radius-md)" }}>
                  <div className="form-label" style={{ color: "var(--accent-purple)" }}>2. Authorizer</div>
                  <div style={{ fontSize: "13px", fontWeight: "600" }}>{matchedRule.authorizer}</div>
                </div>

                <div style={{ padding: "12px", background: "var(--bg-tertiary)", borderRadius: "var(--radius-md)" }}>
                  <div className="form-label" style={{ color: "var(--accent-green)" }}>3. Mutator</div>
                  <div style={{ fontSize: "13px", fontWeight: "600" }}>{matchedRule.mutator}</div>
                </div>
              </div>
            </div>
          ) : (
            <div style={{ padding: "16px", background: "rgba(244, 63, 94, 0.1)", border: "1px solid var(--accent-rose)", borderRadius: "var(--radius-md)", color: "var(--accent-rose)" }}>
              No rule matched this path. Aegis will return <strong>404 Not Found (Zero Trust Default Deny)</strong>.
            </div>
          )}
        </div>

        {/* Declarative Rules File Viewer */}
        <div className="card">
          <div className="card-header">
            <h2 className="card-title">Declarative Rules Definition (`rules/default.rules.yaml`)</h2>
            <span style={{ fontSize: "12px", color: "var(--text-muted)" }}>Hot-reloaded by Aegis</span>
          </div>

          <div className="code-box">
{`version: "v1"
rules:
  - id: "rule-documents-api"
    match:
      url: "/api/v1/documents/<.*>"
      methods: ["GET", "POST"]
    authenticators:
      - handler: "jwt"
        config:
          jwks_url: "http://janus:4444/.well-known/jwks.json"
    authorizers:
      - handler: "nexus_rebac"
        config:
          nexus_grpc_addr: "nexus:50051"
          namespace: "document"
          relation: "viewer"
    mutators:
      - handler: "header"
        config:
          headers:
            X-User-ID: "{{ .Subject }}"
    upstream:
      url: "http://documents-backend:8080"`}
          </div>
        </div>
      </div>
    </>
  );
}
