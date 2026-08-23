# Tasks: Command Dialog State Reset

- [x] 1. Modify `CommandDialog` in `console/src/components/layout/command-dialog.tsx` to handle state reset.
  - [x] 1.1 Create a local `handleOpenChange` wrapper function that accepts an `isOpen` boolean.
  - [x] 1.2 In the wrapper, call the parent `onOpenChange(isOpen)`.
  - [x] 1.3 In the wrapper, if `!isOpen`, call `setQuery("")`.
  - [x] 1.4 Update the `<Dialog>` component to use `handleOpenChange` instead of the raw `onOpenChange` prop.
  - [x] 1.5 Update the `useEffect` keyboard listener to call `handleOpenChange(!open)` instead of `onOpenChange(!open)`.
