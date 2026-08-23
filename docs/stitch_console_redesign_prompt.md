# Context
You are an expert UI/UX Designer and Frontend Engineer building the central Admin Console for "Autorix", an open-source enterprise authentication and authorization platform (similar to Auth0, Ory, or Supabase). 

Autorix is based on a microservices architecture where each engine handles a specific domain. The console must act as the unified control plane for all these engines.

# UI/UX Vibe & Requirements
* **Style:** Enterprise-grade B2B SaaS, modern, clean, high information density. Dark mode by default with stark, high-contrast accents.
* **Layout:** A collapsible left sidebar for navigation. A top header for Workspace/Environment switching (Test vs. Live) and user profile. A main content area for the data.
* **Components:** Use Shadcn UI patterns.
* **Patterns:** 
  - Never use inline creation forms on list pages. Always use slide-overs (Sheets) opening from the right for creating/editing entities.
  - Data tables must look robust (filters, search bars, pagination controls, row-level action dropdowns).
  - Code and JSON must be displayed in monospaced, syntax-highlighted blocks.
  - Interactive "Studios" or "Playgrounds" should be distinct visual panels (cards) placed above or alongside the data tables to test configurations.

# Domain Requirements: The 8 Engines
Please generate the UI screens and components that allow an administrator to manage ALL of the following actions across the 8 engines. 

## 1. Argus (Multi-tenant & Organizations)
**Purpose:** Core hierarchy and billing.
* **Workspaces & Projects:** Switch between organizations, create new projects.
* **Environments:** Toggle between `Test` and `Live` environments.
* **Members Management:** Invite team members, assign RBAC roles (Admin, Developer, Viewer).
* **Settings:** Billing, Usage quotas, API limits.

## 2. Ego (Identities & Authentication)
**Purpose:** User lifecycle and authentication.
* **User Directory:** Data table of all registered users (email, name, status, created date, last login).
* **User Slide-over:** Create/Edit users, force password resets, revoke active sessions.
* **Identity Schema Builder:** A JSON editor to define custom traits/fields (e.g., phone, address) for the tenant.
* **Authentication Providers:** Toggles and configuration forms for Social Logins (Google, GitHub), Passkeys, and MFA methods.

## 3. Nexus (ReBAC Permissions)
**Purpose:** Relationship-based access control (Zanzibar model).
* **Namespaces:** Define object types (e.g., `document`, `folder`) and their relationships (e.g., `owner`, `viewer`).
* **Zanzibar Tuples Table:** Manage relations (Subject, Relation, Object). E.g., `user:123` is `viewer` on `document:456`.
* **Tuple Builder Slide-over:** Form to inject new tuples into the graph.
* **Check Simulator Playground:** A dedicated panel to run live tests: "Does Subject X have Relation Y on Object Z?".

## 4. Vulcan (API Keys & Macaroons)
**Purpose:** Attenuated tokens and machine-to-machine auth.
* **API Keys Table:** List of root API keys with their prefixes (`av_live_...`), scopes, and expiration.
* **Key Builder Slide-over:** Generate new keys, assign owners and base scopes.
* **Attenuation Studio:** A visual playground to take a Root Key and add "Caveats" (e.g., `ip_range = 192.168.1.0/24`, `expires_in = 2h`) to generate a restricted Macaroon offline.

## 5. Themis (ABAC Policies)
**Purpose:** Attribute-based access control using CEL (Common Expression Language).
* **Policies Table:** List of policies with their priority, name, and enabled/disabled toggle.
* **Policy IDE Slide-over:** A monospaced text editor to write CEL expressions (e.g., `request.auth.claims.role == "admin"`), with priority and label settings.
* **Dry-Run Playground:** A dual-pane panel. Left pane: write a mock JSON request payload. Right pane: see if the policy evaluates to `Passed` or `Failed`.

## 6. Aegis (KMS & Secrets)
**Purpose:** Cryptographic key management and vaulting.
* **Keys Table:** Manage Symmetric/Asymmetric encryption keys.
* **Key Actions:** Rotate keys, disable keys, view key versions.
* **Secrets Vault:** Key-value store for environment variables and third-party API keys, masked by default.

## 7. Janus (API Gateway & Routing)
**Purpose:** Edge routing and rate limiting.
* **Routes Table:** Map incoming paths to downstream Autorix engines or custom webhooks.
* **Route Builder Slide-over:** Define path matching, HTTP methods, and attach middlewares.
* **Middlewares:** Configure Rate Limiting (req/sec), CORS, and IP Allowlisting per route.

## 8. Hermes (Audit & Webhooks)
**Purpose:** Observability and event streaming.
* **Audit Logs:** A dense, read-only data table of all system events (e.g., `ego.user.created`, `nexus.tuple.deleted`). Includes timestamp, actor, IP, and raw JSON payload.
* **Webhooks Table:** Configure endpoints to receive Hermes events.
* **Webhook Builder Slide-over:** Define target URL, signing secret, and event subscriptions.

# Deliverables
Please generate the React/Next.js code with Tailwind CSS and Shadcn UI to build this comprehensive dashboard. Focus on the layout shell first, and then the detailed views for Ego, Nexus, Vulcan, and Themis as the priority engines.
