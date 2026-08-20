"use client";

import React, { createContext, useContext, useState, useEffect } from "react";
import { translations, Locale } from "./translations";

type TranslationKey = string;

interface I18nContextType {
  locale: Locale;
  setLocale: (locale: Locale) => void;
  t: (key: TranslationKey, params?: Record<string, string | number>) => string;
  dict: typeof translations.en;
}

const I18nContext = createContext<I18nContextType | undefined>(undefined);

const STORAGE_KEY = "autorix_console_lang";

export function I18nProvider({ children }: { children: React.ReactNode }) {
  const [locale, setLocaleState] = useState<Locale>("en");

  useEffect(() => {
    try {
      const saved = localStorage.getItem(STORAGE_KEY) as Locale | null;
      if (saved && (saved === "en" || saved === "es")) {
        setLocaleState(saved);
      }
    } catch {
      // ignore
    }
  }, []);

  const setLocale = (newLocale: Locale) => {
    setLocaleState(newLocale);
    try {
      localStorage.setItem(STORAGE_KEY, newLocale);
    } catch {
      // ignore
    }
  };

  const currentDict = translations[locale] || translations.en;

  const t = (key: TranslationKey, params?: Record<string, string | number>): string => {
    const keys = key.split(".");
    let current: unknown = currentDict;

    for (const k of keys) {
      if (current && typeof current === "object" && k in current) {
        current = (current as Record<string, unknown>)[k];
      } else {
        // Fallback to English
        let fallback: unknown = translations.en;
        for (const fbKey of keys) {
          if (fallback && typeof fallback === "object" && fbKey in fallback) {
            fallback = (fallback as Record<string, unknown>)[fbKey];
          } else {
            return key;
          }
        }
        current = fallback;
        break;
      }
    }

    if (typeof current !== "string") {
      return key;
    }

    let result = current;
    if (params) {
      Object.entries(params).forEach(([paramKey, paramValue]) => {
        result = result.replace(new RegExp(`{${paramKey}}`, "g"), String(paramValue));
      });
    }

    return result;
  };

  return <I18nContext.Provider value={{ locale, setLocale, t, dict: currentDict }}>{children}</I18nContext.Provider>;
}

export function useI18n() {
  const context = useContext(I18nContext);
  if (!context) {
    throw new Error("useI18n must be used within an I18nProvider");
  }
  return context;
}

export function useTranslation() {
  const { t, locale, setLocale, dict } = useI18n();
  return { t, locale, setLocale, dict };
}
