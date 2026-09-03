# Spec: aegis-rule-management

## Feature: 5-Step Rule Creation Wizard (P6-S5-T6)

### Scenario: Operator creates a proxy rule using the guided wizard
- **Given** the operator is on the Proxy Rules screen (`/proxy-rules`)
- **When** the operator opens the Rule Builder Wizard
- **Then** the wizard guides through 5 sequential steps:
  1. Match (URL pattern with glob/regex support, HTTP methods)
  2. Authenticate (selection and configuration from active authenticators catalogue)
  3. Authorize (selection and configuration from active authorizers catalogue)
  4. Mutate (optional mutator selection and header configs)
  5. Upstream (target upstream URL, optional `strip_prefix`, optional `rewrite`)
- **And** displays a live JSON preview of the synthesized `Rule`
- **When** the operator submits the form
- **Then** `POST /api/proxy-rules` is called with the validated payload
- **And** the rules list refreshes showing the newly created rule

## Feature: Explicit Rule Ordering & Shadowing Detection (P6-S5-T7)

### Scenario: Reordering rules persists explicit priority
- **Given** an ordered list of proxy rules
- **When** the operator adjusts the order of rules (e.g. moving a rule up or down)
- **Then** `PUT /api/proxy-rules/reorder` is invoked with the updated list of rule IDs
- **And** Aegis updates the persisted `order_idx` sequence in Postgres

### Scenario: Shadowing warning when a broad rule precedes a specific rule
- **Given** a broad rule `Rule A` matching methods `GET` and URL `<.*>` at order index `0`
- **And** a more specific rule `Rule B` matching method `GET` and URL `/api/v1/protected/<.*>` at order index `1`
- **When** the rules table renders or after a reorder action
- **Then** the UI displays an amber warning badge on `Rule B` indicating it is shadowed by `Rule A`
- **And** warns that `Rule B` will never be matched because first-match evaluation will always trigger `Rule A` first

## Feature: Rule Version Snapshots & Rollback (P6-S5-T8)

### Scenario: Inspecting versions and rolling back
- **Given** one or more previous rule snapshot versions exist in Aegis
- **When** the operator opens the Version History dialog
- **Then** `GET /api/proxy-rules/versions` retrieves the list of version snapshots with timestamps and rule counts
- **When** the operator selects a previous version and confirms rollback
- **Then** `POST /api/proxy-rules/rollback/[version]` is invoked
- **And** the active rules revert to the snapshot version and the rules table refreshes

## Feature: Deep Dry-Run Pipeline Trace Simulation (P6-S5-T4)

### Scenario: Simulating request execution with step-by-step trace
- **Given** the operator enters a test HTTP method, path, and test headers/session
- **When** the operator clicks "Simulate Pipeline"
- **Then** `POST /api/proxy-rules/test-match` is invoked with the simulation payload
- **And** the response renders a multi-stage trace showing:
  - Matched rule ID
  - Authenticator stage status, extracted subject, and credentials
  - Authorizer stage status and allow/deny verdict
  - Mutator stage status and transformed request headers
  - Upstream target URL with applied prefix stripping and path rewriting
  - Final verdict badge (`ALLOW`, `DENY`, `UNAUTHORIZED`, `ERROR`)
