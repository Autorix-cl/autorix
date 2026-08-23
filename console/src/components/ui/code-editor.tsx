"use client";

import * as React from "react";
import CodeMirror from "@uiw/react-codemirror";
import { json } from "@codemirror/lang-json";
import { oneDark } from "@codemirror/theme-one-dark";
import { cn } from "@/lib/utils";

interface CodeEditorProps {
  value: string;
  onChange: (value: string) => void;
  language?: "json" | "cel"; // We can expand to CEL later
  height?: string;
  className?: string;
  readOnly?: boolean;
}

export function CodeEditor({
  value,
  onChange,
  language = "json",
  height = "300px",
  className,
  readOnly = false,
}: CodeEditorProps) {
  // Currently only setting up JSON. CEL can be supported with custom grammar later.
  const extensions = language === "json" ? [json()] : [];

  return (
    <div className={cn("overflow-hidden rounded-md border", className)}>
      <CodeMirror
        value={value}
        height={height}
        theme={oneDark}
        extensions={extensions}
        onChange={onChange}
        readOnly={readOnly}
        className="text-sm font-mono"
      />
    </div>
  );
}
