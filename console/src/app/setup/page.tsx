"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { ShieldCheck, KeyRound, Terminal, AlertCircle, ArrowRight, CheckCircle2, Lock } from "lucide-react";
import { toast } from "sonner";

export default function SetupPage() {
  const router = useRouter();
  const [checkingStatus, setCheckingStatus] = useState(true);
  const [token, setToken] = useState("");
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let isMounted = true;
    fetch("/api/auth/status")
      .then((res) => (res.ok ? res.json() : null))
      .then((data) => {
        if (!isMounted) return;
        if (data && data.bootstrapped === true) {
          toast.info("Cluster is already initialized. Please authenticate.");
          router.replace("/login");
        } else {
          setCheckingStatus(false);
        }
      })
      .catch(() => {
        if (isMounted) setCheckingStatus(false);
      });
    return () => {
      isMounted = false;
    };
  }, [router]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    if (password !== confirmPassword) {
      setError("Passwords do not match");
      return;
    }

    if (password.length < 8) {
      setError("Password must be at least 8 characters");
      return;
    }

    setLoading(true);
    try {
      const res = await fetch("/api/auth/setup", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          bootstrap_token: token.trim(),
          name: name.trim(),
          email: email.trim().toLowerCase(),
          password,
        }),
      });

      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error || "Failed to complete setup");
      }

      toast.success("Root owner created successfully. Initializing control plane...");
      router.push("/");
      router.refresh();
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : "An unexpected error occurred during bootstrap setup";
      setError(message);
      toast.error(message);
    } finally {
      setLoading(false);
    }
  };

  if (checkingStatus) {
    return (
      <div className="min-h-screen w-full flex items-center justify-center bg-[#080B10] text-slate-500 font-mono text-xs">
        <div className="flex items-center gap-2">
          <div className="h-4 w-4 rounded-full border-2 border-amber-500/40 border-t-amber-400 animate-spin" />
          <span>Verifying cluster bootstrap status...</span>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen w-full flex items-center justify-center p-4 bg-[#080B10] text-[#E2E8F0] relative overflow-hidden selection:bg-amber-500/20 selection:text-amber-300">
      {/* Background ambient security grid */}
      <div className="absolute inset-0 bg-[linear-gradient(to_right,#131B2A15_1px,transparent_1px),linear-gradient(to_bottom,#131B2A15_1px,transparent_1px)] bg-[size:4rem_4rem] pointer-events-none" />
      <div className="absolute top-1/4 -left-32 w-96 h-96 bg-amber-500/5 rounded-full blur-3xl pointer-events-none" />
      <div className="absolute bottom-1/4 -right-32 w-96 h-96 bg-emerald-500/5 rounded-full blur-3xl pointer-events-none" />

      <div className="w-full max-w-xl relative z-10">
        {/* Header Branding */}
        <div className="text-center mb-8">
          <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full border border-amber-500/30 bg-amber-500/10 text-amber-400 text-xs font-mono font-medium mb-4">
            <KeyRound className="w-3.5 h-3.5" />
            INITIAL BOOTSTRAP WIZARD
          </div>
          <h1 className="text-3xl font-semibold tracking-tight text-white flex items-center justify-center gap-3">
            <span className="w-8 h-8 rounded-lg bg-gradient-to-br from-amber-400 to-amber-600 flex items-center justify-center text-black font-mono font-bold text-lg shadow-lg shadow-amber-500/20">
              A
            </span>
            Autorix Control Plane
          </h1>
          <p className="text-sm text-slate-400 mt-2 max-w-md mx-auto">
            Establish the root owner account for your zero-trust IAM cluster. This one-time setup claims the cluster using your server bootstrap token.
          </p>
        </div>

        {/* Setup Card */}
        <div className="rounded-xl border border-slate-800 bg-[#0F141E]/95 backdrop-blur-md p-7 shadow-2xl shadow-black/80">
          <div className="flex items-center gap-3 pb-5 mb-6 border-b border-slate-800/80">
            <div className="w-10 h-10 rounded-lg bg-slate-800/80 flex items-center justify-center text-amber-400">
              <ShieldCheck className="w-5 h-5" />
            </div>
            <div>
              <h2 className="text-base font-medium text-white">Create Root Owner</h2>
              <p className="text-xs text-slate-400">Owner role holds full cluster sovereignty and break-glass capabilities.</p>
            </div>
          </div>

          {error && (
            <div className="mb-6 p-3.5 rounded-lg border border-red-500/30 bg-red-500/10 text-red-300 text-sm flex items-start gap-2.5">
              <AlertCircle className="w-4 h-4 mt-0.5 shrink-0 text-red-400" />
              <span>{error}</span>
            </div>
          )}

          <form onSubmit={handleSubmit} className="space-y-4">
            {/* Bootstrap Token */}
            <div>
              <div className="flex items-center justify-between mb-1.5">
                <label className="text-xs font-medium text-slate-300 flex items-center gap-1.5">
                  <Terminal className="w-3.5 h-3.5 text-amber-400" />
                  Bootstrap Token
                </label>
                <span className="text-[11px] font-mono text-slate-500">From Argus startup logs (abt_...)</span>
              </div>
              <input
                type="text"
                required
                value={token}
                onChange={(e) => setToken(e.target.value)}
                placeholder="abt_0123456789abcdef..."
                className="w-full px-3.5 py-2.5 rounded-lg bg-[#0A0D14] border border-slate-700/80 text-white font-mono text-sm placeholder:text-slate-600 focus:outline-none focus:ring-2 focus:ring-amber-500/40 focus:border-amber-500 transition-all"
              />
            </div>

            {/* Operator Name & Email */}
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3.5">
              <div>
                <label className="block text-xs font-medium text-slate-300 mb-1.5">Full Name</label>
                <input
                  type="text"
                  required
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="Senior Admin"
                  className="w-full px-3.5 py-2.5 rounded-lg bg-[#0A0D14] border border-slate-700/80 text-white text-sm placeholder:text-slate-600 focus:outline-none focus:ring-2 focus:ring-amber-500/40 focus:border-amber-500 transition-all"
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-slate-300 mb-1.5">Work Email</label>
                <input
                  type="email"
                  required
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="admin@autorix.internal"
                  className="w-full px-3.5 py-2.5 rounded-lg bg-[#0A0D14] border border-slate-700/80 text-white text-sm placeholder:text-slate-600 focus:outline-none focus:ring-2 focus:ring-amber-500/40 focus:border-amber-500 transition-all"
                />
              </div>
            </div>

            {/* Password & Confirm */}
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3.5">
              <div>
                <label className="block text-xs font-medium text-slate-300 mb-1.5 flex items-center gap-1.5">
                  <Lock className="w-3.5 h-3.5 text-slate-400" />
                  Master Password
                </label>
                <input
                  type="password"
                  required
                  minLength={8}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="••••••••••••"
                  className="w-full px-3.5 py-2.5 rounded-lg bg-[#0A0D14] border border-slate-700/80 text-white text-sm placeholder:text-slate-600 focus:outline-none focus:ring-2 focus:ring-amber-500/40 focus:border-amber-500 transition-all"
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-slate-300 mb-1.5">Confirm Password</label>
                <input
                  type="password"
                  required
                  minLength={8}
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  placeholder="••••••••••••"
                  className="w-full px-3.5 py-2.5 rounded-lg bg-[#0A0D14] border border-slate-700/80 text-white text-sm placeholder:text-slate-600 focus:outline-none focus:ring-2 focus:ring-amber-500/40 focus:border-amber-500 transition-all"
                />
              </div>
            </div>

            {/* Security Notice */}
            <div className="p-3 rounded-lg bg-slate-900/60 border border-slate-800 text-xs text-slate-400 space-y-1">
              <div className="flex items-center gap-1.5 text-slate-300 font-medium">
                <CheckCircle2 className="w-3.5 h-3.5 text-emerald-400" />
                Argon2id Salted Hashes & Instant Revocation
              </div>
              <p>Credentials are hashed with 64MB memory cost parameters matching Ego engine specifications.</p>
            </div>

            {/* Submit Action */}
            <button
              type="submit"
              disabled={loading}
              className="w-full py-3 px-4 rounded-lg bg-gradient-to-r from-amber-500 to-amber-600 hover:from-amber-400 hover:to-amber-500 text-black font-semibold text-sm transition-all shadow-lg shadow-amber-500/25 flex items-center justify-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer mt-2"
            >
              {loading ? (
                <span>Initializing Owner Account...</span>
              ) : (
                <>
                  <span>Initialize Cluster Owner</span>
                  <ArrowRight className="w-4 h-4" />
                </>
              )}
            </button>
          </form>
        </div>

        {/* Footer info */}
        <div className="text-center mt-6 text-xs text-slate-500">
          Autorix Security Architecture · Control Plane Identity Layer
        </div>
      </div>
    </div>
  );
}
