## Implementation Progress

**Change**: command-dialog-reset
**Mode**: Standard

### Completed Tasks
- [x] 1. Modify `CommandDialog` in `console/src/components/layout/command-dialog.tsx` to handle state reset.
  - [x] 1.1 Create a local `handleOpenChange` wrapper function that accepts an `isOpen` boolean.
  - [x] 1.2 In the wrapper, call the parent `onOpenChange(isOpen)`.
  - [x] 1.3 In the wrapper, if `!isOpen`, call `setQuery("")`.
  - [x] 1.4 Update the `<Dialog>` component to use `handleOpenChange` instead of the raw `onOpenChange` prop.
  - [x] 1.5 Update the `useEffect` keyboard listener to call `handleOpenChange(!open)` instead of `onOpenChange(!open)`.

### Files Changed
| File | Action | What Was Done |
|------|--------|---------------|
| `console/src/components/layout/command-dialog.tsx` | Modified | Implemented `handleOpenChange` callback wrapped in `React.useCallback`, mapped to Dialog's `onOpenChange` and keyboard handler, and reset `query` state when closed. |

### Deviations from Design
Added `React.useCallback` wrapping around `handleOpenChange` to satisfy `exhaustive-deps` linter rules because it is provided as a dependency to the keyboard `useEffect` hook.

### Issues Found
None.

### Remaining Tasks
None.

### Status
5/5 tasks complete. Ready for verify.
