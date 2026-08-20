import "../styles/globals.css";
import { AppShell } from "@/components/layout/app-shell";
import { AppQueryProvider } from "@/lib/query/provider";
import { AuthProvider } from "@/lib/auth/use-permission";
import { CapabilityProvider } from "@/lib/capabilities/capability-context";
import { ThemeProvider } from "@/lib/theme-provider";
import { Toaster } from "sonner";

import { EnvironmentProvider } from "@/lib/environment/environment-context";

export const metadata = {
  title: "Autorix Console | Enterprise Zero Trust IAM Platform",
  description: "Administrative console and developer portal for the 6 Autorix IAM Engines",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body className="font-sans antialiased bg-background text-foreground selection:bg-primary/20 selection:text-primary">
        <AppQueryProvider>
          <ThemeProvider defaultTheme="dark">
            <EnvironmentProvider>
              <AuthProvider>
                <CapabilityProvider>
                  <AppShell>{children}</AppShell>
                  <Toaster theme="dark" position="top-right" richColors closeButton />
                </CapabilityProvider>
              </AuthProvider>
            </EnvironmentProvider>
          </ThemeProvider>
        </AppQueryProvider>
      </body>
    </html>
  );
}
