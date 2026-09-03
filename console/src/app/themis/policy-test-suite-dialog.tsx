"use client";

import * as React from "react";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import {
  FlaskConical,
  Play,
  CheckCircle2,
  XCircle,
  Plus,
  Trash2,
  Loader2,
  AlertCircle,
} from "lucide-react";
import { toast } from "sonner";
import type { PolicyFixture, TestSuiteResult } from "@/lib/api/schemas/themis";

interface PolicyTestSuiteDialogProps {
  policyId: string | null;
  policyName: string | null;
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
}

export function PolicyTestSuiteDialog({
  policyId,
  policyName,
  isOpen,
  onOpenChange,
}: PolicyTestSuiteDialogProps) {
  const [fixtures, setFixtures] = React.useState<PolicyFixture[]>([]);
  const [isLoading, setIsLoading] = React.useState(false);
  const [isRunning, setIsRunning] = React.useState(false);
  const [suiteResult, setSuiteResult] = React.useState<TestSuiteResult | null>(null);

  // New fixture state
  const [isAddingFixture, setIsAddingFixture] = React.useState(false);
  const [newFixtureName, setNewFixtureName] = React.useState("");
  const [newFixtureExpected, setNewFixtureExpected] = React.useState(true);
  const [newFixturePayload, setNewFixturePayload] = React.useState('{\n  "role": "admin",\n  "mfa": true\n}');

  const loadFixtures = React.useCallback(async () => {
    if (!policyId) return;
    setIsLoading(true);
    try {
      const res = await fetch(`/api/themis/policies/${policyId}/fixtures`);
      if (!res.ok) throw new Error("Failed to load fixtures");
      const data = await res.json();
      setFixtures(Array.isArray(data) ? data : []);
    } catch {
      toast.error("Failed to load test fixtures");
    } finally {
      setIsLoading(false);
    }
  }, [policyId]);

  React.useEffect(() => {
    if (isOpen && policyId) {
      loadFixtures();
      setSuiteResult(null);
      setIsAddingFixture(false);
    }
  }, [isOpen, policyId, loadFixtures]);

  const handleRunSuite = async () => {
    if (!policyId) return;
    setIsRunning(true);
    try {
      const res = await fetch(`/api/themis/policies/${policyId}/test-suite`, {
        method: "POST",
      });
      if (!res.ok) {
        const err = await res.json();
        throw new Error(err.error || "Failed to execute test suite");
      }
      const data: TestSuiteResult = await res.json();
      setSuiteResult(data);
      if (data.all_passed) {
        toast.success(`All ${data.total_tests} tests passed!`);
      } else {
        toast.error(`${data.failed_tests} out of ${data.total_tests} tests failed.`);
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Test suite failed");
    } finally {
      setIsRunning(false);
    }
  };

  const handleAddFixture = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!policyId || !newFixtureName.trim()) return;

    let parsedPayload: Record<string, unknown> = {};
    try {
      parsedPayload = JSON.parse(newFixturePayload);
    } catch {
      toast.error("Payload must be valid JSON");
      return;
    }

    try {
      const res = await fetch(`/api/themis/policies/${policyId}/fixtures`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          name: newFixtureName.trim(),
          expected_result: newFixtureExpected,
          payload: parsedPayload,
        }),
      });
      if (!res.ok) throw new Error("Failed to save fixture");
      toast.success("Test fixture added");
      setNewFixtureName("");
      setIsAddingFixture(false);
      loadFixtures();
    } catch {
      toast.error("Failed to create test fixture");
    }
  };

  const handleDeleteFixture = async (fixtureId: string) => {
    if (!policyId) return;
    try {
      const res = await fetch(`/api/themis/policies/${policyId}/fixtures/${fixtureId}`, {
        method: "DELETE",
      });
      if (!res.ok) throw new Error("Failed to delete fixture");
      toast.success("Fixture removed");
      loadFixtures();
    } catch {
      toast.error("Failed to delete test fixture");
    }
  };

  return (
    <Dialog open={isOpen} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl max-h-[85vh] flex flex-col">
        <DialogHeader>
          <div className="flex items-center justify-between pr-4">
            <DialogTitle className="flex items-center gap-2">
              <FlaskConical className="w-5 h-5 text-primary" />
              Test Suite · {policyName || policyId}
            </DialogTitle>
            <Button
              size="sm"
              onClick={handleRunSuite}
              disabled={isRunning || fixtures.length === 0}
              className="gap-1.5 text-xs"
            >
              {isRunning ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <Play className="w-3.5 h-3.5" />}
              Run All Tests
            </Button>
          </div>
          <DialogDescription className="text-xs">
            Manage test scenarios and verify policy logic against assertions before production rollout.
          </DialogDescription>
        </DialogHeader>

        <div className="flex-1 overflow-y-auto space-y-4 pt-2 pr-1">
          {/* Test Suite Summary Banner */}
          {suiteResult && (
            <div
              className={`p-3 rounded-lg border flex items-center justify-between text-xs ${
                suiteResult.all_passed
                  ? "bg-emerald-500/10 border-emerald-500/30 text-emerald-500"
                  : "bg-destructive/10 border-destructive/30 text-destructive"
              }`}
            >
              <div className="flex items-center gap-2 font-medium">
                {suiteResult.all_passed ? (
                  <CheckCircle2 className="w-4 h-4" />
                ) : (
                  <XCircle className="w-4 h-4" />
                )}
                <span>
                  {suiteResult.all_passed
                    ? "All Test Scenarios Passed"
                    : `${suiteResult.failed_tests} Scenario(s) Failed`}
                </span>
              </div>
              <div className="flex items-center gap-3 text-[11px]">
                <span>Total: {suiteResult.total_tests}</span>
                <span>Passed: {suiteResult.passed_tests}</span>
                <span>Failed: {suiteResult.failed_tests}</span>
              </div>
            </div>
          )}

          {/* Test Fixture Creation Form */}
          {isAddingFixture ? (
            <form onSubmit={handleAddFixture} className="p-3 border rounded-lg bg-card/60 space-y-3 text-xs">
              <div className="flex items-center justify-between">
                <span className="font-semibold text-foreground">New Test Fixture</span>
                <Button variant="ghost" size="sm" onClick={() => setIsAddingFixture(false)} className="h-6 text-[11px]">
                  Cancel
                </Button>
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1">
                  <Label className="text-[11px]">Fixture Name</Label>
                  <Input
                    placeholder="e.g., Admin MFA authorized"
                    value={newFixtureName}
                    onChange={(e) => setNewFixtureName(e.target.value)}
                    className="h-7 text-xs"
                    required
                  />
                </div>
                <div className="flex items-center gap-2 pt-5">
                  <Label className="text-[11px]">Expected Evaluation:</Label>
                  <div className="flex items-center gap-1.5">
                    <Switch
                      checked={newFixtureExpected}
                      onCheckedChange={setNewFixtureExpected}
                    />
                    <Badge variant={newFixtureExpected ? "default" : "secondary"} className="text-[10px]">
                      {newFixtureExpected ? "True (Allow)" : "False (Deny)"}
                    </Badge>
                  </div>
                </div>
              </div>

              <div className="space-y-1">
                <Label className="text-[11px]">Input Payload (JSON)</Label>
                <textarea
                  value={newFixturePayload}
                  onChange={(e) => setNewFixturePayload(e.target.value)}
                  rows={4}
                  className="w-full font-mono text-[11px] p-2 rounded border bg-muted/20"
                />
              </div>

              <Button type="submit" size="sm" className="h-7 text-xs">
                Save Fixture
              </Button>
            </form>
          ) : (
            <div className="flex justify-between items-center">
              <span className="text-xs font-semibold text-foreground">
                Fixtures ({fixtures.length})
              </span>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setIsAddingFixture(true)}
                className="h-7 text-xs gap-1"
              >
                <Plus className="w-3.5 h-3.5" /> Add Scenario
              </Button>
            </div>
          )}

          {/* Fixtures List */}
          {isLoading ? (
            <div className="flex items-center justify-center py-6 text-xs text-muted-foreground gap-2">
              <Loader2 className="w-4 h-4 animate-spin" />
              Loading fixtures...
            </div>
          ) : fixtures.length === 0 ? (
            <div className="text-center py-6 text-xs text-muted-foreground border rounded-lg border-dashed">
              No test scenarios attached yet. Add one to create regression tests.
            </div>
          ) : (
            <div className="space-y-2">
              {fixtures.map((fix) => {
                const runResult = suiteResult?.results.find((r) => r.fixture_id === fix.id);

                return (
                  <div
                    key={fix.id}
                    className="p-3 rounded-lg border bg-card/60 flex items-start justify-between gap-3 text-xs"
                  >
                    <div className="space-y-1.5 flex-1">
                      <div className="flex items-center gap-2">
                        <span className="font-semibold text-foreground">{fix.name}</span>
                        <Badge variant="outline" className="text-[10px]">
                          Expected: {fix.expected_result ? "Allow" : "Deny"}
                        </Badge>
                        {runResult && (
                          <Badge
                            variant={runResult.passed ? "outline" : "destructive"}
                            className={`text-[10px] ${
                              runResult.passed
                                ? "text-emerald-500 border-emerald-500/30"
                                : ""
                            }`}
                          >
                            {runResult.passed ? (
                              <CheckCircle2 className="w-3 h-3 mr-1" />
                            ) : (
                              <AlertCircle className="w-3 h-3 mr-1" />
                            )}
                            {runResult.passed ? "Passed" : `Failed (got ${runResult.actual_result})`}
                          </Badge>
                        )}
                      </div>

                      <pre className="font-mono text-[10px] bg-muted/40 p-2 rounded text-muted-foreground overflow-x-auto max-h-20">
                        {JSON.stringify(fix.payload, null, 2)}
                      </pre>
                    </div>

                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-7 w-7 text-muted-foreground hover:text-destructive"
                      onClick={() => handleDeleteFixture(fix.id)}
                    >
                      <Trash2 className="w-3.5 h-3.5" />
                    </Button>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
