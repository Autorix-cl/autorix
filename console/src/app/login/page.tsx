"use client";

import { useState, Suspense } from "react";
import { useSearchParams } from "next/navigation";
import { Lock, ShieldAlert, KeyRound, ArrowRight, Building2, AlertCircle, Eye, EyeOff } from "lucide-react";
import { toast } from "sonner";

export default function LoginPage() {
  return (
    <Suspense fallback={<div className="min-h-screen bg-[#080B10] flex items-center justify-center text-slate-500 font-mono text-xs">Authenticating...</div>}>
      <LoginForm />
    </Suspense>
  );
}

function LoginForm() {
  const searchParams = useSearchParams();
  const from = searchParams.get("from") || "/";

  const [activeTab, setActiveTab] = useState<"local" | "sso">("local");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleLocalLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setLoading(true);

    try {
      const res = await fetch("/api/auth/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          email: email.trim().toLowerCase(),
          password,
        }),
      });

      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error || "Authentication failed");
      }

      toast.success(`Welcome back, ${data.operator?.name || "Operator"}`);
      window.location.href = from;
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : "Invalid email or password";
      setError(message);
      toast.error(message);
    } finally {
      setLoading(false);
    }
  };

  const handleSSOLogin = () => {
    toast.info("Connecting to registered Janus/Ego OpenID Connect provider...");
    window.location.href = "/api/auth/sso/login";
  };

  return (
    <div className="min-h-screen w-full flex items-center justify-center p-4 bg-[#080B10] text-[#E2E8F0] relative overflow-hidden selection:bg-amber-500/20 selection:text-amber-300">
      {/* Ambient background glow */}
      <div className="absolute inset-0 bg-[linear-gradient(to_right,#131B2A15_1px,transparent_1px),linear-gradient(to_bottom,#131B2A15_1px,transparent_1px)] bg-[size:4rem_4rem] pointer-events-none" />
      <div className="absolute top-1/3 -left-40 w-96 h-96 bg-amber-500/5 rounded-full blur-3xl pointer-events-none" />
      <div className="absolute bottom-1/3 -right-40 w-96 h-96 bg-blue-500/5 rounded-full blur-3xl pointer-events-none" />

      <div className="w-full max-w-md relative z-10">
        {/* Brand Header */}
        <div className="text-center mb-8">
          <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full border border-slate-800 bg-[#0F141E] text-slate-300 text-xs font-mono mb-4">
            <Lock className="w-3.5 h-3.5 text-amber-400" />
            ZERO TRUST CONTROL PLANE
          </div>
          <h1 className="text-3xl font-semibold tracking-tight text-white flex items-center justify-center gap-3">
            <span className="w-8 h-8 rounded-lg bg-gradient-to-br from-amber-400 to-amber-600 flex items-center justify-center text-black font-mono font-bold text-lg shadow-lg shadow-amber-500/20">
              A
            </span>
            Autorix Console
          </h1>
          <p className="text-sm text-slate-400 mt-2">
            Authenticate to access the administrative control plane
          </p>
        </div>

        {/* Card */}
        <div className="rounded-xl border border-slate-800 bg-[#0F141E]/95 backdrop-blur-md p-7 shadow-2xl shadow-black/80">
          {/* Method Tabs */}
          <div className="grid grid-cols-2 gap-1 p-1 bg-[#090C12] rounded-lg border border-slate-800 mb-6">
            <button
              type="button"
              onClick={() => setActiveTab("local")}
              className={`py-2 px-3 rounded-md text-xs font-medium transition-all flex items-center justify-center gap-2 ${
                activeTab === "local"
                  ? "bg-[#161C28] text-white shadow-sm border border-slate-700/60"
                  : "text-slate-400 hover:text-slate-200"
              }`}
            >
              <KeyRound className="w-3.5 h-3.5 text-amber-400" />
              Break-Glass Operator
            </button>
            <button
              type="button"
              onClick={() => setActiveTab("sso")}
              className={`py-2 px-3 rounded-md text-xs font-medium transition-all flex items-center justify-center gap-2 ${
                activeTab === "sso"
                  ? "bg-[#161C28] text-white shadow-sm border border-slate-700/60"
                  : "text-slate-400 hover:text-slate-200"
              }`}
            >
              <Building2 className="w-3.5 h-3.5 text-blue-400" />
              Enterprise SSO
            </button>
          </div>

          {error && (
            <div className="mb-6 p-3.5 rounded-lg border border-red-500/30 bg-red-500/10 text-red-300 text-sm flex items-start gap-2.5">
              <AlertCircle className="w-4 h-4 mt-0.5 shrink-0 text-red-400" />
              <span>{error}</span>
            </div>
          )}

          {activeTab === "local" ? (
            <form onSubmit={handleLocalLogin} className="space-y-4">
              <div>
                <label className="block text-xs font-medium text-slate-300 mb-1.5">Operator Email</label>
                <input
                  type="email"
                  required
                  autoFocus
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="operator@autorix.internal"
                  className="w-full px-3.5 py-2.5 rounded-lg bg-[#0A0D14] border border-slate-700/80 text-white text-sm placeholder:text-slate-600 focus:outline-none focus:ring-2 focus:ring-amber-500/40 focus:border-amber-500 transition-all"
                />
              </div>

              <div>
                <div className="flex items-center justify-between mb-1.5">
                  <label className="text-xs font-medium text-slate-300">Master Password</label>
                </div>
                <div className="relative">
                  <input
                    type={showPassword ? "text" : "password"}
                    required
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    placeholder="••••••••••••"
                    className="w-full px-3.5 py-2.5 pr-10 rounded-lg bg-[#0A0D14] border border-slate-700/80 text-white text-sm placeholder:text-slate-600 focus:outline-none focus:ring-2 focus:ring-amber-500/40 focus:border-amber-500 transition-all font-mono"
                  />
                  <button
                    type="button"
                    onClick={() => setShowPassword(!showPassword)}
                    className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-500 hover:text-slate-300 transition-colors"
                  >
                    {showPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                  </button>
                </div>
              </div>

              <div className="p-3 rounded-lg bg-amber-500/5 border border-amber-500/20 text-xs text-amber-300/80 flex items-start gap-2">
                <ShieldAlert className="w-4 h-4 shrink-0 text-amber-400 mt-0.5" />
                <span>Local operator logins are strictly audited and retain access during upstream SSO recovery.</span>
              </div>

              <button
                type="submit"
                disabled={loading}
                className="w-full py-3 px-4 rounded-lg bg-gradient-to-r from-amber-500 to-amber-600 hover:from-amber-400 hover:to-amber-500 text-black font-semibold text-sm transition-all shadow-lg shadow-amber-500/20 flex items-center justify-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer mt-2"
              >
                {loading ? (
                  <span>Authenticating...</span>
                ) : (
                  <>
                    <span>Authenticate Session</span>
                    <ArrowRight className="w-4 h-4" />
                  </>
                )}
              </button>
            </form>
          ) : (
            <div className="space-y-4 py-2">
              <div className="text-center p-6 border border-dashed border-slate-700/80 rounded-lg bg-slate-900/30">
                <Building2 className="w-8 h-8 text-blue-400 mx-auto mb-3" />
                <h3 className="text-sm font-medium text-white mb-1">Corporate Single Sign-On</h3>
                <p className="text-xs text-slate-400 mb-4">
                  Sign in through your registered corporate IdP via Autorix Janus (OIDC) or Hermes (SAML).
                </p>
                <button
                  type="button"
                  onClick={handleSSOLogin}
                  className="w-full py-2.5 px-4 rounded-lg bg-[#162032] hover:bg-[#1E2C44] border border-blue-500/30 text-blue-300 font-medium text-sm transition-all flex items-center justify-center gap-2 cursor-pointer shadow-lg shadow-blue-500/10"
                >
                  <span>Continue with Enterprise SSO</span>
                  <ArrowRight className="w-4 h-4" />
                </button>
              </div>
            </div>
          )}
        </div>

        {/* Security Footer */}
        <div className="text-center mt-6 text-xs text-slate-500 flex items-center justify-center gap-4">
          <span>Encrypted Session</span>
          <span>·</span>
          <span>Zero Trust PEP</span>
          <span>·</span>
          <span>Argon2id Hash</span>
        </div>
      </div>
    </div>
  );
}
