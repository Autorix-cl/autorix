"use client";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { FileText } from "lucide-react";
export function LogViewer(props: { onSelectTrace?: (traceId: string) => void }) { void props; return <Card><CardHeader><CardTitle className="text-sm flex gap-2"><FileText className="h-4 w-4" />Structured logs</CardTitle><CardDescription>No log aggregation backend is configured.</CardDescription></CardHeader><CardContent className="text-xs text-muted-foreground">Connect a supported log backend before querying logs. Console intentionally does not retain fabricated or in-memory log entries.</CardContent></Card>; }
