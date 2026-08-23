# Spec: console-navigation

## Feature: Command Dialog State Management

### Scenario: Dialog clears search query upon closing
- **Given** the Command Dialog is open
- **And** the user has typed "nexus" into the search input
- **When** the dialog is closed (via item selection, Esc key, or clicking outside)
- **Then** the search query state is immediately reset to empty
- **And** when the dialog is opened again, the search input is empty
