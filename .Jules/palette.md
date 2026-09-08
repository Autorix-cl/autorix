## 2024-03-24 - Missing ARIA Labels on Icon Buttons
**Learning:** Found several icon-only buttons across the app without `aria-label` attributes, which creates an accessibility issue for screen readers. Some buttons use `sr-only` text, which works, but many didn't have either.
**Action:** Always ensure icon-only `<Button>` components have an explicit `aria-label` attribute if they do not contain nested screen-reader text (`<span className="sr-only">`).
