import Link from "next/link";
import { ShieldX, ArrowLeft, Lock } from "lucide-react";

export default function ForbiddenPage() {
  return (
    <div className="min-h-screen w-full flex items-center justify-center p-4 bg-[#080B10] text-[#E2E8F0] relative">
      <div className="w-full max-w-md text-center">
        <div className="w-16 h-16 rounded-2xl bg-red-500/10 border border-red-500/30 flex items-center justify-center mx-auto mb-6 text-red-400 shadow-xl shadow-red-500/10">
          <ShieldX className="w-8 h-8" />
        </div>

        <div className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full border border-red-500/20 bg-red-500/5 text-red-400 text-xs font-mono mb-3">
          <Lock className="w-3.5 h-3.5" />
          HTTP 403 · ACCESS FORBIDDEN
        </div>

        <h1 className="text-2xl font-bold text-white mb-2">Insufficient Role Privileges</h1>
        <p className="text-sm text-slate-400 mb-8 max-w-sm mx-auto">
          Your operator role does not have the required permissions to perform this action or view this resource. Contact your cluster owner to adjust your role assignments.
        </p>

        <div className="flex items-center justify-center gap-3">
          <Link
            href="/"
            className="px-4 py-2.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-white text-xs font-medium transition-all flex items-center gap-2"
          >
            <ArrowLeft className="w-4 h-4" />
            Back to Fleet Overview
          </Link>
        </div>
      </div>
    </div>
  );
}
