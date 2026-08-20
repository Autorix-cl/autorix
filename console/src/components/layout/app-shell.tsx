"use client";

import * as React from "react";
import { I18nProvider } from "@/lib/i18n";
import { TooltipProvider } from "@/components/ui/tooltip";
import { Sidebar } from "@/components/layout/sidebar";
import { Header } from "@/components/layout/header";
import { CommandDialog } from "@/components/layout/command-dialog";

export function AppShell({ children }: { children: React.ReactNode }) {
  const [searchOpen, setSearchOpen] = React.useState(false);

  return (
    <I18nProvider>
      <TooltipProvider>
        <div className="flex h-screen w-full overflow-hidden bg-background text-foreground">
          {/* Collapsible/Fixed Enterprise Sidebar */}
          <Sidebar />

          {/* Main Content Area */}
          <div className="flex flex-1 flex-col overflow-hidden">
            <Header onOpenSearch={() => setSearchOpen(true)} />

            <main className="flex-1 overflow-y-auto px-6 py-6 lg:px-8 subtle-mesh-bg">
              <div className="mx-auto max-w-7xl space-y-6">{children}</div>
            </main>
          </div>

          <CommandDialog open={searchOpen} onOpenChange={setSearchOpen} />
        </div>
      </TooltipProvider>
    </I18nProvider>
  );
}
