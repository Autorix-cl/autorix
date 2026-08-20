export function formatDateTime(
  date: Date | string | number,
  locale: "en" | "es" = "en",
  timezone?: string
): string {
  const d = typeof date === "object" ? date : new Date(date);
  return new Intl.DateTimeFormat(locale === "es" ? "es-ES" : "en-US", {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    timeZone: timezone,
  }).format(d);
}

export function formatRelativeTime(
  date: Date | string | number,
  locale: "en" | "es" = "en"
): string {
  const d = typeof date === "object" ? date : new Date(date);
  const now = Date.now();
  const diffSeconds = Math.round((d.getTime() - now) / 1000);
  const diffMinutes = Math.round(diffSeconds / 60);
  const diffHours = Math.round(diffMinutes / 60);
  const diffDays = Math.round(diffHours / 24);

  const rtf = new Intl.RelativeTimeFormat(locale === "es" ? "es-ES" : "en-US", {
    numeric: "auto",
  });

  if (Math.abs(diffSeconds) < 60) {
    return rtf.format(diffSeconds, "second");
  }
  if (Math.abs(diffMinutes) < 60) {
    return rtf.format(diffMinutes, "minute");
  }
  if (Math.abs(diffHours) < 24) {
    return rtf.format(diffHours, "hour");
  }
  return rtf.format(diffDays, "day");
}
