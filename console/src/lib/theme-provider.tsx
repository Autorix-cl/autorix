"use client";

import * as React from "react";

type Theme = "dark" | "light" | "system";

interface ThemeProviderProps {
  children: React.ReactNode;
  defaultTheme?: Theme;
  storageKey?: string;
}

interface ThemeProviderState {
  theme: Theme;
  setTheme: (theme: Theme) => void;
  resolvedTheme: "dark" | "light";
}

const initialState: ThemeProviderState = {
  theme: "dark",
  setTheme: () => null,
  resolvedTheme: "dark",
};

const ThemeProviderContext = React.createContext<ThemeProviderState>(initialState);

function safeGetTheme(storageKey: string): Theme | null {
  try {
    const saved = localStorage.getItem(storageKey);
    if (saved === "dark" || saved === "light" || saved === "system") {
      return saved;
    }
    return null;
  } catch {
    return null;
  }
}

function safeSetTheme(storageKey: string, value: Theme): void {
  try {
    localStorage.setItem(storageKey, value);
  } catch {
    // Ignore storage access errors in restricted browsing modes
  }
}

export function ThemeProvider({
  children,
  defaultTheme = "dark",
  storageKey = "autorix-theme",
  ...props
}: ThemeProviderProps) {
  const [theme, setThemeState] = React.useState<Theme>(defaultTheme);
  const [resolvedTheme, setResolvedTheme] = React.useState<"dark" | "light">("dark");

  React.useEffect(() => {
    const savedTheme = safeGetTheme(storageKey);
    if (savedTheme) {
      setThemeState(savedTheme);
    }
  }, [storageKey]);

  React.useEffect(() => {
    const root = window.document.documentElement;

    const applyResolvedTheme = (resolved: "dark" | "light") => {
      root.classList.remove("light", "dark");
      root.classList.add(resolved);
      root.style.colorScheme = resolved;
      setResolvedTheme(resolved);
    };

    if (theme === "system") {
      const mediaQuery = window.matchMedia("(prefers-color-scheme: dark)");
      applyResolvedTheme(mediaQuery.matches ? "dark" : "light");

      const handleChange = (event: MediaQueryListEvent) => {
        applyResolvedTheme(event.matches ? "dark" : "light");
      };
      mediaQuery.addEventListener("change", handleChange);
      return () => {
        mediaQuery.removeEventListener("change", handleChange);
      };
    }

    applyResolvedTheme(theme);
  }, [theme]);

  const setTheme = React.useCallback(
    (newTheme: Theme) => {
      safeSetTheme(storageKey, newTheme);
      setThemeState(newTheme);
    },
    [storageKey],
  );

  const value = React.useMemo(
    () => ({ theme, setTheme, resolvedTheme }),
    [theme, setTheme, resolvedTheme],
  );

  return (
    <ThemeProviderContext.Provider {...props} value={value}>
      {children}
    </ThemeProviderContext.Provider>
  );
}

export const useTheme = () => {
  const context = React.useContext(ThemeProviderContext);
  if (context === undefined) throw new Error("useTheme must be used within a ThemeProvider");
  return context;
};
