import "../styles/globals.css";
import Link from "next/link";
import { 
  Shield, 
  Users, 
  KeyRound, 
  Network, 
  Layers, 
  Building2, 
  LayoutDashboard,
  Terminal,
  Activity
} from "lucide-react";

export const metadata = {
  title: "Autorix Console | Zero Trust IAM Platform",
  description: "Administrative console and developer portal for Autorix IAM Suite",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body>
        <div className="app-container">
          {/* Sidebar */}
          <aside className="sidebar">
            <div style={{ padding: "24px 20px", borderBottom: "1px solid var(--border-subtle)", display: "flex", alignItems: "center", gap: "10px" }}>
              <div style={{ background: "linear-gradient(135deg, #3b82f6, #8b5cf6)", width: "32px", height: "32px", borderRadius: "8px", display: "flex", alignItems: "center", justifyContent: "center" }}>
                <Shield size={18} color="#fff" />
              </div>
              <div>
                <h1 style={{ fontSize: "16px", fontWeight: "700", letterSpacing: "-0.02em" }}>AUTORIX</h1>
                <span style={{ fontSize: "11px", color: "var(--text-muted)", textTransform: "uppercase", letterSpacing: "0.05em" }}>IAM Suite v1.0</span>
              </div>
            </div>

            <nav style={{ padding: "16px 12px", display: "flex", flexDirection: "column", gap: "4px", flex: 1 }}>
              <div style={{ fontSize: "10px", fontWeight: "700", color: "var(--text-muted)", padding: "8px 12px", textTransform: "uppercase", letterSpacing: "0.08em" }}>
                Overview
              </div>
              <Link href="/" style={{ display: "flex", alignItems: "center", gap: "10px", padding: "10px 12px", borderRadius: "var(--radius-md)", color: "var(--text-primary)", fontSize: "13px", fontWeight: "500" }}>
                <LayoutDashboard size={16} color="var(--accent-blue)" />
                <span>Dashboard</span>
              </Link>

              <div style={{ fontSize: "10px", fontWeight: "700", color: "var(--text-muted)", padding: "16px 12px 8px 12px", textTransform: "uppercase", letterSpacing: "0.08em" }}>
                Core Engines
              </div>
              <Link href="/identities" style={{ display: "flex", alignItems: "center", gap: "10px", padding: "10px 12px", borderRadius: "var(--radius-md)", color: "var(--text-secondary)", fontSize: "13px" }}>
                <Users size={16} />
                <span>Ego (Identities)</span>
              </Link>

              <Link href="/permissions" style={{ display: "flex", alignItems: "center", gap: "10px", padding: "10px 12px", borderRadius: "var(--radius-md)", color: "var(--text-secondary)", fontSize: "13px" }}>
                <Network size={16} />
                <span>Nexus (Zanzibar ReBAC)</span>
              </Link>

              <Link href="/oauth2" style={{ display: "flex", alignItems: "center", gap: "10px", padding: "10px 12px", borderRadius: "var(--radius-md)", color: "var(--text-secondary)", fontSize: "13px" }}>
                <KeyRound size={16} />
                <span>Janus (OAuth2 / OIDC)</span>
              </Link>

              <Link href="/proxy-rules" style={{ display: "flex", alignItems: "center", gap: "10px", padding: "10px 12px", borderRadius: "var(--radius-md)", color: "var(--text-secondary)", fontSize: "13px" }}>
                <Shield size={16} />
                <span>Aegis (Zero Trust Proxy)</span>
              </Link>

              <Link href="/api-keys" style={{ display: "flex", alignItems: "center", gap: "10px", padding: "10px 12px", borderRadius: "var(--radius-md)", color: "var(--text-secondary)", fontSize: "13px" }}>
                <Layers size={16} />
                <span>Vulcan (Macaroons)</span>
              </Link>

              <Link href="/enterprise" style={{ display: "flex", alignItems: "center", gap: "10px", padding: "10px 12px", borderRadius: "var(--radius-md)", color: "var(--text-secondary)", fontSize: "13px" }}>
                <Building2 size={16} />
                <span>Hermes (SAML & SCIM)</span>
              </Link>
            </nav>

            <div style={{ padding: "16px", borderTop: "1px solid var(--border-subtle)", background: "var(--bg-tertiary)" }}>
              <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: "8px" }}>
                <div style={{ display: "flex", alignItems: "center", gap: "8px" }}>
                  <Activity size={14} color="var(--accent-green)" />
                  <span style={{ fontSize: "12px", fontWeight: "600" }}>Cluster Status</span>
                </div>
                <span className="badge badge-green" style={{ fontSize: "10px", padding: "2px 6px" }}>6/6 Online</span>
              </div>
              <p style={{ fontSize: "11px", color: "var(--text-muted)" }}>Zero Trust Perimeter Active</p>
            </div>
          </aside>

          {/* Main Content Area */}
          <main className="main-content">
            {children}
          </main>
        </div>
      </body>
    </html>
  );
}
