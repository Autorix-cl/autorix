/* eslint-disable */
import { ThemisPoliciesTable } from "./policies-table";
import { DryRunPlayground } from "./dry-run-playground";
import { PolicyBuilderSheet } from "./policy-builder-sheet";


export default function ThemisPage() {
  return (
    <div className="flex-1 space-y-4 p-8 pt-6 overflow-auto">
      <div className="flex items-center justify-between space-y-2">
        <div>
          <h2 className="text-3xl font-bold tracking-tight">Themis (ABAC Policies)</h2>
          <p className="text-muted-foreground">
            Attribute-based access control engine. Write policies in Common Expression Language (CEL).
          </p>
        </div>
        <div className="flex items-center space-x-2">
          <PolicyBuilderSheet />
        </div>
      </div>
      
      <div className="space-y-4">
        <ThemisPoliciesTable />
        <DryRunPlayground />
      </div>
    </div>
  );
}
