# Change: Command Dialog State Reset

## Context
In the Autorix Console, the `CommandDialog` (triggered via `⌘K`) is used to quickly navigate between engines and settings. Currently, if a user types a query and then closes the dialog (either by selecting an item or pressing `Esc`), the query state is preserved. When the dialog is reopened, the previous search term is still present, requiring the user to manually backspace before starting a new search.

## Objective
Ensure the `CommandDialog` always opens with a clean slate by resetting the search query when the dialog is closed.

## Scope
- **In Scope**: Modifying the `CommandDialog` component's state management in the `console` app.
- **Out of Scope**: Adding new search functionality, redesigning the dialog, or persisting search history.

## Capabilities
1. **console-navigation**: The `CommandDialog` must clear its search query state whenever it transitions to a closed state.
