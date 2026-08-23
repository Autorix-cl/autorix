import * as React from "react";
import { Button } from "@/components/ui/button";
import { CodeEditor } from "@/components/ui/code-editor";
import { Shield, KeyRound, Loader2 } from "lucide-react";

export interface MfaPanelProps {
  factors: string[];
  onGenerateRecovery: () => Promise<string[]>;
}

export function MfaPanel({ factors, onGenerateRecovery }: MfaPanelProps) {
  const [isGenerating, setIsGenerating] = React.useState(false);
  const [recoveryCodes, setRecoveryCodes] = React.useState<string | null>(null);

  const handleGenerate = async () => {
    setIsGenerating(true);
    try {
      const codes = await onGenerateRecovery();
      setRecoveryCodes(codes.join("\n"));
    } finally {
      setIsGenerating(false);
    }
  };

  return (
    <div className="space-y-4">
      <div className="rounded-md border p-4">
        <h4 className="text-sm font-medium mb-3 flex items-center gap-2">
          <Shield className="h-4 w-4" />
          Enrolled Factors
        </h4>
        {factors.length === 0 ? (
          <p className="text-sm text-muted-foreground">No MFA factors enrolled.</p>
        ) : (
          <ul className="space-y-2">
            {factors.map((factor, idx) => (
              <li key={idx} className="flex items-center gap-2 text-sm bg-muted/50 p-2 rounded">
                <span className="h-2 w-2 rounded-full bg-green-500" />
                {factor}
              </li>
            ))}
          </ul>
        )}
      </div>

      <div className="rounded-md border p-4">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h4 className="text-sm font-medium mb-1">Recovery Codes</h4>
            <p className="text-xs text-muted-foreground">Generate single-use backup codes for emergencies.</p>
          </div>
          <Button variant="outline" size="sm" onClick={handleGenerate} disabled={isGenerating}>
            {isGenerating ? <Loader2 className="h-4 w-4 mr-2 animate-spin" /> : <KeyRound className="h-4 w-4 mr-2" />}
            Generate Recovery Codes
          </Button>
        </div>

        {recoveryCodes && (
          <div className="mt-4">
            <p className="text-xs font-medium text-destructive mb-2">
              Warning: These codes will only be shown once. Copy them now.
            </p>
            <CodeEditor
              value={recoveryCodes}
              onChange={() => {}} // readOnly in practice
              language="json"
              height="150px"
            />
          </div>
        )}
      </div>
    </div>
  );
}
