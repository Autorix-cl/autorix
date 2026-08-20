"use client";

import * as React from "react";
import Link from "next/link";
import {
  Sparkles,
  KeyRound,
  ArrowRight,
  CheckCircle2,
  Radio,
} from "lucide-react";
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";

export default function WelcomeOnboardingPage() {
  const [firstRegistrationDetected, setFirstRegistrationDetected] = React.useState(false);

  React.useEffect(() => {
    const interval = setInterval(async () => {
      try {
        const res = await fetch("/api/fleet/instances");
        if (res.ok) {
          const data = await res.json();
          const list = Array.isArray(data) ? data : data.items || [];
          if (list.length > 0) {
            setFirstRegistrationDetected(true);
          }
        }
      } catch {
        // Continue polling
      }
    }, 5000);

    return () => clearInterval(interval);
  }, []);

  return (
    <div className="space-y-8 max-w-4xl mx-auto py-8">
      <div className="text-center space-y-3">
        <div className="inline-flex p-3 rounded-2xl bg-primary/10 border border-primary/20 text-primary mb-2">
          <Sparkles className="h-8 w-8" />
        </div>
        <h1 className="text-3xl font-bold tracking-tight text-foreground">Welcome to Autorix Control Plane</h1>
        <p className="text-sm text-muted-foreground max-w-md mx-auto">
          Get started by enrolling your first IAM engine into the unified control plane.
        </p>
      </div>

      {/* Guided 3-Step Path */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <Card className="border-border/80 bg-card">
          <CardHeader className="p-5 pb-3">
            <div className="flex items-center gap-3">
              <span className="flex h-7 w-7 items-center justify-center rounded-full bg-primary/10 text-primary font-bold font-mono text-xs">
                1
              </span>
              <CardTitle className="text-sm font-semibold">Select Environment</CardTitle>
            </div>
            <CardDescription className="text-xs mt-2">
              Your default environment is <span className="font-mono text-foreground font-semibold">prod</span>. Use the header dropdown anytime to switch contexts.
            </CardDescription>
          </CardHeader>
          <CardContent className="p-5 pt-0">
            <Badge variant="outline" className="text-[10px] font-mono text-emerald-400">
              Configured
            </Badge>
          </CardContent>
        </Card>

        <Card className="border-border/80 bg-card">
          <CardHeader className="p-5 pb-3">
            <div className="flex items-center gap-3">
              <span className="flex h-7 w-7 items-center justify-center rounded-full bg-primary/10 text-primary font-bold font-mono text-xs">
                2
              </span>
              <CardTitle className="text-sm font-semibold">Mint Enrollment Token</CardTitle>
            </div>
            <CardDescription className="text-xs mt-2">
              Generate a high-entropy <span className="font-mono text-foreground">aet_</span> token scoped to your engine type.
            </CardDescription>
          </CardHeader>
          <CardContent className="p-5 pt-0">
            <Link href="/fleet/enroll">
              <Button size="sm" variant="outline" className="w-full text-xs gap-1.5">
                <KeyRound className="h-3.5 w-3.5 text-primary" /> Mint Token
              </Button>
            </Link>
          </CardContent>
        </Card>

        <Card className="border-border/80 bg-card">
          <CardHeader className="p-5 pb-3">
            <div className="flex items-center gap-3">
              <span className="flex h-7 w-7 items-center justify-center rounded-full bg-primary/10 text-primary font-bold font-mono text-xs">
                3
              </span>
              <CardTitle className="text-sm font-semibold">Launch Engine</CardTitle>
            </div>
            <CardDescription className="text-xs mt-2">
              Pass <span className="font-mono text-foreground">AUTORIX_ENROLLMENT_TOKEN</span> to your container or binary.
            </CardDescription>
          </CardHeader>
          <CardContent className="p-5 pt-0">
            <div className="flex items-center gap-2 text-xs font-mono text-muted-foreground">
              <Radio className="h-3.5 w-3.5 text-amber-400 animate-pulse" />
              <span>Listening for heartbeats...</span>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Real-time Registration Detection Banner */}
      {firstRegistrationDetected && (
        <Card className="border-emerald-500/40 bg-emerald-950/20 p-4 animate-in fade-in">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <CheckCircle2 className="h-6 w-6 text-emerald-400" />
              <div>
                <h4 className="text-sm font-bold text-foreground">First Engine Registration Detected!</h4>
                <p className="text-xs text-muted-foreground">Your cluster is live and receiving telemetry.</p>
              </div>
            </div>
            <Link href="/fleet">
              <Button size="sm" className="text-xs gap-1.5">
                Go to Fleet Dashboard <ArrowRight className="h-3.5 w-3.5" />
              </Button>
            </Link>
          </div>
        </Card>
      )}
    </div>
  );
}
