"use client";
import * as React from "react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Target } from "lucide-react";
export function SLODashboard() {
  return <Card className="border-border"><CardHeader><CardTitle className="text-sm flex gap-2"><Target className="h-4 w-4" />SLOs and error budgets</CardTitle><CardDescription>SLO recording rules are not configured.</CardDescription></CardHeader><CardContent className="text-xs text-muted-foreground">Configure Prometheus recording rules to make SLO data available. Autorix does not calculate or display estimated budgets.</CardContent></Card>;
}
