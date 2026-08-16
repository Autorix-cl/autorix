"use client";

import { useState } from "react";
import { Network, Play, CheckCircle2, XCircle, Zap, ShieldAlert } from "lucide-react";

export default function PermissionsPage() {
  const [namespace, setNamespace] = useState("document");
  const [object, setObject] = useState("financial_report_2026");
  const [relation, setRelation] = useState("viewer");
  const [subjectId, setSubjectId] = useState("alice");
  const [requestContext, setRequestContext] = useState('{\n  "ip": "192.168.1.100",\n  "amount": 450\n}');

  const [evaluating, setEvaluating] = useState(false);
  const [result, setResult] = useState<{
    allowed: boolean;
    reason: string;
    latencyMs: number;
    path: string;
  } | null>({
    allowed: true,
    reason: "direct match (tuple: document:financial_report_2026#viewer@user:alice)",
    latencyMs: 1.2,
    path: "user:alice -> direct relation -> document:financial_report_2026",
  });

  const handleSimulate = (e: React.FormEvent) => {
    e.preventDefault();
    setEvaluating(true);

    setTimeout(() => {
      let allowed = true;
      let reason = "direct relation match";
      let path = `user:${subjectId} -> direct relation -> ${namespace}:${object}#${relation}`;

      // Simulate CEL caveat check
      try {
        const parsed = JSON.parse(requestContext);
        if (parsed.ip && parsed.ip.startsWith("10.0")) {
          allowed = false;
          reason = "CEL caveat failed: ctx.ip != office_network";
        }
      } catch (err) {
        // ignore parse
      }

      setResult({
        allowed: allowed,
        reason: reason,
        latencyMs: Number((Math.random() * 2 + 0.8).toFixed(2)),
        path: path,
      });
      setEvaluating(false);
    }, 400);
  };

  return (
    <>
      <header className="page-header">
        <div>
          <h1 style={{ fontSize: "20px", fontWeight: "700", letterSpacing: "-0.02em" }}>Autorix Nexus: Zanzibar ReBAC & CEL Studio</h1>
          <p style={{ fontSize: "13px", color: "var(--text-secondary)", marginTop: "2px" }}>
            High-speed permission evaluator combining Google Zanzibar relation trees and Google CEL dynamic caveats (:50051 gRPC).
          </p>
        </div>
        <span className="badge badge-purple">
          <Zap size={12} /> In-Memory Goroutine Traversal
        </span>
      </header>

      <div className="page-body">
        <div className="grid-2">
          {/* Query Form */}
          <div className="card">
            <h2 className="card-title" style={{ marginBottom: "16px", display: "flex", alignItems: "center", gap: "8px" }}>
              <Play size={16} color="var(--accent-purple)" /> Permission Check Simulator
            </h2>

            <form onSubmit={handleSimulate}>
              <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: "12px" }}>
                <div className="form-group">
                  <label className="form-label">Namespace</label>
                  <input 
                    type="text" 
                    className="form-input" 
                    value={namespace} 
                    onChange={(e) => setNamespace(e.target.value)} 
                    required 
                  />
                </div>
                <div className="form-group">
                  <label className="form-label">Object ID</label>
                  <input 
                    type="text" 
                    className="form-input" 
                    value={object} 
                    onChange={(e) => setObject(e.target.value)} 
                    required 
                  />
                </div>
              </div>

              <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: "12px" }}>
                <div className="form-group">
                  <label className="form-label">Relation</label>
                  <select 
                    className="form-select" 
                    value={relation} 
                    onChange={(e) => setRelation(e.target.value)}
                  >
                    <option value="viewer">viewer</option>
                    <option value="editor">editor</option>
                    <option value="owner">owner</option>
                    <option value="member">member</option>
                  </select>
                </div>
                <div className="form-group">
                  <label className="form-label">Subject ID</label>
                  <input 
                    type="text" 
                    className="form-input" 
                    value={subjectId} 
                    onChange={(e) => setSubjectId(e.target.value)} 
                    required 
                  />
                </div>
              </div>

              <div className="form-group">
                <label className="form-label">Dynamic Request Context (CEL ABAC Variables)</label>
                <textarea 
                  className="form-textarea" 
                  value={requestContext} 
                  onChange={(e) => setRequestContext(e.target.value)} 
                  rows={4}
                />
              </div>

              <button 
                type="submit" 
                className="btn btn-primary" 
                style={{ width: "100%", background: "var(--accent-purple)", borderColor: "var(--accent-purple)" }}
                disabled={evaluating}
              >
                <Play size={14} /> {evaluating ? "Evaluating Graph in Memory..." : "Evaluate Permission (gRPC Check)"}
              </button>
            </form>
          </div>

          {/* Results Panel */}
          <div className="card">
            <h2 className="card-title" style={{ marginBottom: "16px" }}>Evaluation Result</h2>

            {result ? (
              <div>
                <div style={{ 
                  padding: "20px", 
                  borderRadius: "var(--radius-md)", 
                  backgroundColor: result.allowed ? "rgba(16, 185, 129, 0.1)" : "rgba(244, 63, 94, 0.1)",
                  border: `1px solid ${result.allowed ? "var(--accent-green)" : "var(--accent-rose)"}`,
                  display: "flex",
                  alignItems: "center",
                  gap: "16px",
                  marginBottom: "20px"
                }}>
                  {result.allowed ? (
                    <CheckCircle2 size={36} color="var(--accent-green)" />
                  ) : (
                    <XCircle size={36} color="var(--accent-rose)" />
                  )}
                  <div>
                    <div style={{ fontSize: "20px", fontWeight: "700", color: result.allowed ? "var(--accent-green)" : "var(--accent-rose)" }}>
                      {result.allowed ? "ACCESS ALLOWED" : "ACCESS DENIED"}
                    </div>
                    <div style={{ fontSize: "12px", color: "var(--text-muted)", marginTop: "2px" }}>
                      Resolved in <strong style={{ color: "var(--text-primary)" }}>{result.latencyMs} ms</strong> via concurrent short-circuiting
                    </div>
                  </div>
                </div>

                <div className="form-group">
                  <label className="form-label">Decision Reason</label>
                  <div className="code-box">{result.reason}</div>
                </div>

                <div className="form-group">
                  <label className="form-label">Resolution Path</label>
                  <div className="code-box" style={{ color: "var(--accent-purple)" }}>{result.path}</div>
                </div>
              </div>
            ) : (
              <p style={{ color: "var(--text-muted)" }}>Execute a check query to simulate graph evaluation.</p>
            )}
          </div>
        </div>
      </div>
    </>
  );
}
