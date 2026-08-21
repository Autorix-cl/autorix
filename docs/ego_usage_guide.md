# Manage enterprise identities and sessions with Autorix Ego

Autorix Ego is an enterprise identity and user lifecycle engine that provides dynamic JSON Schema-driven identity traits, memory-hard Argon2id password hashing, RFC 6238 TOTP multi-factor authentication (MFA), recovery codes, and high-performance active session management.

### Quick path

1. Start Ego on port `4433`.
2. Register a user by sending a `POST /self-service/registration` request with dynamic traits and password.
3. Authenticate the user with `POST /self-service/login` using the identifier and password.
4. Manage the resulting session with `GET /sessions/whoami` or revoke it with `POST /self-service/logout`.

### Details

#### Dynamic JSON Schema Traits
Ego does not hardcode rigid user columns. Instead, user attributes are modeled as Dynamic Traits validated against JSON Schema (Draft 7):
* **Standard fields**: `email`, `name`, `username`.
* **Custom enterprise fields**: `department`, `employee_id`, `cost_center`, `security_clearance`, `phone_number`.

#### Cryptographic Security
Ego strictly adheres to OWASP password storage recommendations with Argon2id parameters (m=64MB, t=3, p=4) to eliminate GPU/ASIC acceleration attacks. It supports RFC 6238 TOTP (6-digit codes refreshed every 30s) and single-use recovery codes.

#### User Self-Service APIs
* **Registration**: `POST /self-service/registration` registers a new identity and generates an active session. A state-machine flow (Ory Kratos parity) is also available via `GET /self-service/registration/browser`. WebAuthn is supported.
* **Login**: `POST /self-service/login` authenticates an identity.
* **Session Validation**: `GET /sessions/whoami` resolves the caller identity from the `autorix_session_token` cookie or Bearer header.
* **Logout**: `POST /self-service/logout` revokes the active session token.

#### Administrative APIs
* **Identity Management**: Use `GET /admin/identities`, `PATCH /admin/identities/{id}`, and `DELETE /admin/identities/{id}` to list, modify, or permanently delete identities.
* **Password Reset**: `POST /admin/identities/{id}/credentials/reset-password` resets directly, or `POST /admin/identities/{id}/recovery-link` generates a recovery link.
* **MFA & Sessions**: Use `GET /admin/sessions` and `DELETE /admin/sessions/{id}` to view and revoke sessions, or `DELETE /admin/identities/{id}/mfa` to revoke TOTP configuration.
* **Schemas Management**: Use `GET /admin/schemas` and `POST /admin/schemas` to register and list JSON Schema definitions.

### Checklist

- [ ] Define the enterprise JSON Schema traits (e.g., `corp_employee_v1`).
- [ ] Use `/self-service/registration` or state-machine endpoints to onboard users.
- [ ] Configure Argon2id password hashing parameters according to your security needs.
- [ ] Test the TOTP MFA and recovery code enrollment.

### Next step

- Read the [Console usage guide](./console_usage_guide.md) for visualizing identities.
- Explore the [Hermes federation guide](./hermes_usage_guide.md) to sync users from external IdPs (Okta, Entra ID) to Ego.
