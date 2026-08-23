# Design: Command Dialog State Reset

## Approach
We will intercept the `onOpenChange` callback provided to the `<Dialog>` component in `console/src/components/layout/command-dialog.tsx`. When the dialog is instructed to close (`open === false`), we will clear the search query.

## File Changes
1. `console/src/components/layout/command-dialog.tsx`
   - Wrap the `onOpenChange` handler passed to `<Dialog>` to also call `setQuery("")` when `open` is false.
   - We will also ensure the `handleSelect` function doesn't need to manually clear the query, since calling `onOpenChange(false)` inside it will automatically trigger the cleanup.

## Decisions
- **Why wrap `onOpenChange` instead of just modifying `handleSelect`?**
  If we only cleared the query in `handleSelect`, pressing `Esc` or clicking the dialog backdrop would close it without clearing the state. By tying the state reset to the dialog's closed lifecycle event (`onOpenChange(false)`), we guarantee a clean slate regardless of how the user dismisses the dialog.
