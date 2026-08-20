"use client";

import * as React from "react";
import { usePathname } from "next/navigation";
import { I18nProvider } from "@/lib/i18n";
import { TooltipProvider } from "@/components/ui/tooltip";
import { Sidebar } from "@/components/layout/sidebar";
import { Header } from "@/components/layout/header";
import { CommandDialog } from "@/components/layout/command-dialog";

const STANDALONE_ROUTES = ["/login", "/setup", "/session-expired", "/403"];

export function AppShell({ children }: { children: React.ReactNode }) {
  const [searchOpen, setSearchOpen] = React.useState(false);
  const pathname = usePathname();

  const isStandalone = STANDALONE_ROUTES.some(
    (route) => pathname === route || pathname?.startsWith(route + "/")
  );

  if (isStandalone) {
    return (
      <I18nProvider>
        <TooltipProvider>{children}</TooltipProvider>
      </I18nProvider>
    );
  }

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
