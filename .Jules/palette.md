## 2024-03-24 - Tooltips on Icon-Only Buttons
**Learning:** Found that when wrapping icon-only buttons with Radix UI Tooltip components (e.g. `TooltipTrigger asChild`), we still need to keep the explicit `aria-label` directly on the underlying `<Button>`. The Tooltip provides `aria-describedby` but doesn't replace the primary accessible name.
**Action:** Added tooltips to icon-only "Copy" buttons in the OAuth2 Client and Vulcan Key Builder dialogues to provide visual feedback (especially for success states like "Copied!"), while preserving `aria-label="Copy to clipboard"` on the Button element to avoid accessibility regressions.

## 2023-10-27 - Standardized Retry Buttons
**Learning:** Found that some state components (`ErrorState`, `NotConnectedState`) were using native `<button>` tags with custom Tailwind classes instead of the design system's `<Button>` component. This led to missing keyboard focus visible states and slight visual inconsistencies with the rest of the application.
**Action:** Standardized "Retry" buttons across state components to use `<Button variant="outline" size="sm">` to ensure consistent accessibility (focus rings) and visual design patterns.

## 2024-05-18 - Discoverability for Micro-Actions
**Learning:** Icon-only action buttons (like "Move Up", "Move Down", "Simulate", "Delete" in the Proxy Rules pipeline view) benefit greatly from rich tooltips (`Tooltip` from `@radix-ui/react-tooltip`) rather than native `title` attributes. Tooltips can be styled consistently with the application and provide a better UX, especially when the icon alone doesn't convey the full context (e.g. `Play` for "Simulate"). We must continue to preserve `aria-label` for screen reader accessibility. Also note that disabled Shadcn UI buttons may not trigger hover tooltips, so wrapping them in an accessible generic element might be necessary if tooltips are needed on disabled items.
**Action:** Replaced native `title` attributes with rich Radix UI tooltips on the proxy rules row actions, preserving `aria-label`s.
