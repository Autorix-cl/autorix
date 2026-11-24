## 2024-03-24 - Tooltips on Icon-Only Buttons
**Learning:** Found that when wrapping icon-only buttons with Radix UI Tooltip components (e.g. `TooltipTrigger asChild`), we still need to keep the explicit `aria-label` directly on the underlying `<Button>`. The Tooltip provides `aria-describedby` but doesn't replace the primary accessible name.
**Action:** Added tooltips to icon-only "Copy" buttons in the OAuth2 Client and Vulcan Key Builder dialogues to provide visual feedback (especially for success states like "Copied!"), while preserving `aria-label="Copy to clipboard"` on the Button element to avoid accessibility regressions.

## 2023-10-27 - Standardized Retry Buttons
**Learning:** Found that some state components (`ErrorState`, `NotConnectedState`) were using native `<button>` tags with custom Tailwind classes instead of the design system's `<Button>` component. This led to missing keyboard focus visible states and slight visual inconsistencies with the rest of the application.
**Action:** Standardized "Retry" buttons across state components to use `<Button variant="outline" size="sm">` to ensure consistent accessibility (focus rings) and visual design patterns.
