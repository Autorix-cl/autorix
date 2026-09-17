## 2023-10-27 - Standardized Retry Buttons
**Learning:** Found that some state components (`ErrorState`, `NotConnectedState`) were using native `<button>` tags with custom Tailwind classes instead of the design system's `<Button>` component. This led to missing keyboard focus visible states and slight visual inconsistencies with the rest of the application.
**Action:** Standardized "Retry" buttons across state components to use `<Button variant="outline" size="sm">` to ensure consistent accessibility (focus rings) and visual design patterns.
