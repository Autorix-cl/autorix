"use client";

import { SERVICES_CONFIG } from "@/lib/config";
import Link from "next/link";
import { 
  Shield, 
  Users, 
  KeyRound, 
  Network, 
  Layers, 
  Building2, 
  ArrowUpRight,
  Server,
  Zap,
  Lock,
  CheckCircle2
} from "lucide-react";

export default function DashboardPage() {
  return (
    <>
      <header className="page-header">
        <div>
          <h1 style={{ fontSize: "20px", fontWeight: "700", letterSpacing: "-0.02em" }}>Ecosystem Overview</h1>
          <p style={{ fontSize: "13px", color: "var(--text-secondary)", marginTop: "2px" }}>
            Real-time status, telemetry and configuration for the 6 Autorix IAM Engines.
          </p>
        </div>
        <div style={{ display: "flex", gap: "12px" }}>
          <span className="badge badge-green">
            <CheckCircle2 size={12} />
            PostgreSQL Cluster Connected
          </span>
        </div>
      </header>

      <div className="page-body">
        {/* Metric Cards */}
        <div className="grid-3" style={{ marginBottom: "24px" }}>
          <div className="card" style={{ marginBottom: 0 }}>
            <div className="card-header">
              <span className="form-label" style={{ marginBottom: 0 }}>Active Microservices</span>
              <Server size={16} color="var(--accent-blue)" />
            </div>
            <div style={{ fontSize: "28px", fontWeight: "700", color: "var(--text-primary)" }}>6 / 6</div>
            <p style={{ fontSize: "12px", color: "var(--text-muted)", marginTop: "4px" }}>Nexus, Ego, Janus, Aegis, Vulcan, Hermes</p>
          </div>

          <div className="card" style={{ marginBottom: 0 }}>
            <div className="card-header">
              <span className="form-label" style={{ marginBottom: 0 }}>Perimeter Security</span>
              <Shield size={16} color="var(--accent-green)" />
            </div>
            <div style={{ fontSize: "28px", fontWeight: "700", color: "var(--text-primary)" }}>Zero Trust</div>
            <p style={{ fontSize: "12px", color: "var(--text-muted)", marginTop: "4px" }}>Aegis PEP Reverse Proxy :4455 Active</p>
          </div>

          <div className="card" style={{ marginBottom: 0 }}>
            <div className="card-header">
              <span className="form-label" style={{ marginBottom: 0 }}>ReBAC Latency</span>
              <Zap size={16} color="var(--accent-purple)" />
            </div>
            <div style={{ fontSize: "28px", fontWeight: "700", color: "var(--text-primary)" }}>&lt; 2.5 ms</div>
            <p style={{ fontSize: "12px", color: "var(--text-muted)", marginTop: "4px" }}>In-memory concurrent Goroutine graph traversal</p>
          </div>
        </div>

        {/* Services Grid */}
        <h2 style={{ fontSize: "16px", fontWeight: "600", marginBottom: "16px" }}>Core Engine Fleet</h2>
        
        <div className="grid-3" style={{ marginBottom: "32px" }}>
          {/* Nexus */}
          <div className="card" style={{ marginBottom: 0 }}>
            <div className="card-header">
              <div style={{ display: "flex", alignItems: "center", gap: "10px" }}>
                <Network size={18} color="var(--accent-purple)" />
                <h3 className="card-title">{SERVICES_CONFIG.nexus.name}</h3>
              </div>
              <span className="badge badge-purple">{SERVICES_CONFIG.nexus.protocol}</span>
            </div>
            <p style={{ fontSize: "13px", color: "var(--text-secondary)", marginBottom: "16px" }}>
              {SERVICES_CONFIG.nexus.role}
            </p>
            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", paddingTop: "12px", borderTop: "1px solid var(--border-subtle)" }}>
              <span style={{ fontSize: "12px", fontFamily: "var(--font-mono)", color: "var(--text-muted)" }}>Port :{SERVICES_CONFIG.nexus.port}</span>
              <Link href="/permissions" className="btn btn-secondary" style={{ padding: "4px 10px", fontSize: "12px" }}>
                Graph Studio <ArrowUpRight size={12} />
              </Link>
            </div>
          </div>

          {/* Ego */}
          <div className="card" style={{ marginBottom: 0 }}>
            <div className="card-header">
              <div style={{ display: "flex", alignItems: "center", gap: "10px" }}>
                <Users size={18} color="var(--accent-blue)" />
                <h3 className="card-title">{SERVICES_CONFIG.ego.name}</h3>
              </div>
              <span className="badge badge-blue">{SERVICES_CONFIG.ego.protocol}</span>
            </div>
            <p style={{ fontSize: "13px", color: "var(--text-secondary)", marginBottom: "16px" }}>
              {SERVICES_CONFIG.ego.role}
            </p>
            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", paddingTop: "12px", borderTop: "1px solid var(--border-subtle)" }}>
              <span style={{ fontSize: "12px", fontFamily: "var(--font-mono)", color: "var(--text-muted)" }}>Port :{SERVICES_CONFIG.ego.port}</span>
              <Link href="/identities" className="btn btn-secondary" style={{ padding: "4px 10px", fontSize: "12px" }}>
                Identities <ArrowUpRight size={12} />
              </Link>
            </div>
          </div>

          {/* Janus */}
          <div className="card" style={{ marginBottom: 0 }}>
            <div className="card-header">
              <div style={{ display: "flex", alignItems: "center", gap: "10px" }}>
                <KeyRound size={18} color="var(--accent-amber)" />
                <h3 className="card-title">{SERVICES_CONFIG.janus.name}</h3>
              </div>
              <span className="badge badge-amber">{SERVICES_CONFIG.janus.protocol}</span>
            </div>
            <p style={{ fontSize: "13px", color: "var(--text-secondary)", marginBottom: "16px" }}>
              {SERVICES_CONFIG.janus.role}
            </p>
            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", paddingTop: "12px", borderTop: "1px solid var(--border-subtle)" }}>
              <span style={{ fontSize: "12px", fontFamily: "var(--font-mono)", color: "var(--text-muted)" }}>Port :{SERVICES_CONFIG.janus.port}</span>
              <Link href="/oauth2" className="btn btn-secondary" style={{ padding: "4px 10px", fontSize: "12px" }}>
                OAuth2 & JWKS <ArrowUpRight size={12} />
              </Link>
            </div>
          </div>

          {/* Aegis */}
          <div className="card" style={{ marginBottom: 0 }}>
            <div className="card-header">
              <div style={{ display: "flex", alignItems: "center", gap: "10px" }}>
                <Shield size={18} color="var(--accent-green)" />
                <h3 className="card-title">{SERVICES_CONFIG.aegis.name}</h3>
              </div>
              <span className="badge badge-green">{SERVICES_CONFIG.aegis.protocol}</span>
            </div>
            <p style={{ fontSize: "13px", color: "var(--text-secondary)", marginBottom: "16px" }}>
              {SERVICES_CONFIG.aegis.role}
            </p>
            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", paddingTop: "12px", borderTop: "1px solid var(--border-subtle)" }}>
              <span style={{ fontSize: "12px", fontFamily: "var(--font-mono)", color: "var(--text-muted)" }}>Port :{SERVICES_CONFIG.aegis.port}</span>
              <Link href="/proxy-rules" className="btn btn-secondary" style={{ padding: "4px 10px", fontSize: "12px" }}>
                Proxy Rules <ArrowUpRight size={12} />
              </Link>
            </div>
          </div>

          {/* Vulcan */}
          <div className="card" style={{ marginBottom: 0 }}>
            <div className="card-header">
              <div style={{ display: "flex", alignItems: "center", gap: "10px" }}>
                <Layers size={18} color="var(--accent-cyan)" />
                <h3 className="card-title">{SERVICES_CONFIG.vulcan.name}</h3>
              </div>
              <span className="badge badge-blue">{SERVICES_CONFIG.vulcan.protocol}</span>
            </div>
            <p style={{ fontSize: "13px", color: "var(--text-secondary)", marginBottom: "16px" }}>
              {SERVICES_CONFIG.vulcan.role}
            </p>
            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", paddingTop: "12px", borderTop: "1px solid var(--border-subtle)" }}>
              <span style={{ fontSize: "12px", fontFamily: "var(--font-mono)", color: "var(--text-muted)" }}>Port :{SERVICES_CONFIG.vulcan.port}</span>
              <Link href="/api-keys" className="btn btn-secondary" style={{ padding: "4px 10px", fontSize: "12px" }}>
                API Keys <ArrowUpRight size={12} />
              </Link>
            </div>
          </div>

          {/* Hermes */}
          <div className="card" style={{ marginBottom: 0 }}>
            <div className="card-header">
              <div style={{ display: "flex", alignItems: "center", gap: "10px" }}>
                <Building2 size={18} color="var(--accent-rose)" />
                <h3 className="card-title">{SERVICES_CONFIG.hermes.name}</h3>
              </div>
              <span className="badge badge-amber">{SERVICES_CONFIG.hermes.protocol}</span>
            </div>
            <p style={{ fontSize: "13px", color: "var(--text-secondary)", marginBottom: "16px" }}>
              {SERVICES_CONFIG.hermes.role}
            </p>
            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", paddingTop: "12px", borderTop: "1px solid var(--border-subtle)" }}>
              <span style={{ fontSize: "12px", fontFamily: "var(--font-mono)", color: "var(--text-muted)" }}>Port :{SERVICES_CONFIG.hermes.port}</span>
              <Link href="/enterprise" className="btn btn-secondary" style={{ padding: "4px 10px", fontSize: "12px" }}>
                SAML & SCIM <ArrowUpRight size={12} />
              </Link>
            </div>
          </div>
        </div>
      </div>
    </>
  );
}
