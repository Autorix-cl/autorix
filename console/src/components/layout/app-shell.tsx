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
  const [mobileNavOpen, setMobileNavOpen] = React.useState(false);
  const pathname = usePathname();

  React.useEffect(() => {
    setMobileNavOpen(false);
  }, [pathname]);

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
        <a
          href="#main-content"
          className="sr-only focus:not-sr-only focus:fixed focus:left-4 focus:top-4 focus:z-[100] focus:rounded-md focus:bg-primary focus:px-3 focus:py-2 focus:text-xs focus:font-semibold focus:text-primary-foreground focus:shadow-lg focus:outline-none focus:ring-2 focus:ring-ring"
        >
          Skip to main content
        </a>
        <div className="flex h-screen w-full overflow-hidden bg-background text-foreground">
          {/* Responsive Enterprise Sidebar (desktop persistent + mobile drawer) */}
          <Sidebar mobileOpen={mobileNavOpen} onMobileOpenChange={setMobileNavOpen} />

          {/* Main Content Area */}
          <div className="flex min-w-0 flex-1 flex-col overflow-hidden">
            <Header
              onOpenSearch={() => setSearchOpen(true)}
              onOpenMobileNav={() => setMobileNavOpen(true)}
              mobileNavOpen={mobileNavOpen}
            />

            <main
              id="main-content"
              tabIndex={-1}
              className="flex-1 overflow-y-auto px-4 py-5 sm:px-6 sm:py-6 lg:px-8 subtle-mesh-bg focus:outline-none"
            >
              <div className="mx-auto max-w-7xl space-y-6">{children}</div>
            </main>
          </div>

          <CommandDialog open={searchOpen} onOpenChange={setSearchOpen} />
        </div>
      </TooltipProvider>
    </I18nProvider>
  );
}
