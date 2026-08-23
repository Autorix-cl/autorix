import * as React from "react";
import { Button } from "@/components/ui/button";
import { UserX, X } from "lucide-react";

export interface BulkActionBarProps {
  selectedCount: number;
  onSuspend: () => void;
  onClearSelection: () => void;
}

export function BulkActionBar({ selectedCount, onSuspend, onClearSelection }: BulkActionBarProps) {
  if (selectedCount === 0) return null;

  return (
    <div className="fixed bottom-6 left-1/2 -translate-x-1/2 z-50 flex items-center gap-4 bg-foreground text-background px-4 py-3 rounded-lg shadow-xl animate-in slide-in-from-bottom-5">
      <div className="flex items-center gap-2 border-r border-background/20 pr-4">
        <span className="text-sm font-medium">{selectedCount} users selected</span>
        <Button variant="ghost" size="icon" className="h-6 w-6 text-background hover:bg-background/20 hover:text-background" onClick={onClearSelection}>
          <X className="h-4 w-4" />
        </Button>
      </div>
      <div className="flex items-center gap-2">
        <Button variant="destructive" size="sm" onClick={onSuspend}>
          <UserX className="h-4 w-4 mr-2" />
          Suspend
        </Button>
      </div>
    </div>
  );
}
