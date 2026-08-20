"use client";

import * as React from "react";
import { Check, Copy, Terminal } from "lucide-react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";

interface CodeBlockProps extends React.HTMLAttributes<HTMLDivElement> {
  code: string;
  language?: string;
  title?: string;
  showLineNumbers?: boolean;
}

export function CodeBlock({
  code,
  language = "json",
  title,
  showLineNumbers = false,
  className,
  ...props
}: CodeBlockProps) {
  const [copied, setCopied] = React.useState(false);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(code);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // ignore
    }
  };

  const lines = code.trim().split("\n");

  return (
    <div
      className={cn(
        "relative rounded-lg border border-border/80 bg-slate-950/90 dark:bg-black/60 shadow-inner overflow-hidden font-mono text-xs",
        className,
      )}
      {...props}
    >
      {/* Header bar */}
      <div className="flex items-center justify-between border-b border-border/50 bg-muted/30 px-3 py-1.5 text-muted-foreground">
        <div className="flex items-center gap-2">
          <Terminal className="h-3.5 w-3.5 text-muted-foreground/70" />
          <span className="text-[11px] font-medium text-foreground/80">{title || language.toUpperCase()}</span>
        </div>
        <Button
          variant="ghost"
          size="icon-sm"
          onClick={handleCopy}
          className="h-6 w-6 text-muted-foreground hover:text-foreground"
          title="Copy code"
        >
          {copied ? <Check className="h-3.5 w-3.5 text-emerald-400" /> : <Copy className="h-3.5 w-3.5" />}
        </Button>
      </div>

      {/* Code content */}
      <div className="overflow-x-auto p-3 text-slate-200">
        {showLineNumbers ? (
          <table className="w-full border-collapse">
            <tbody>
              {lines.map((line, idx) => (
                <tr key={idx} className="leading-5">
                  <td className="w-8 select-none pr-3 text-right text-slate-600">{idx + 1}</td>
                  <td className="whitespace-pre">{line}</td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : (
          <pre className="whitespace-pre-wrap break-all leading-relaxed">
            <code>{code}</code>
          </pre>
        )}
      </div>
    </div>
  );
}
