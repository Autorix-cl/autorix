# Autorix Hermes: Enterprise SAML 2.0 & SCIM 2.0 Federation Manual

**Autorix Hermes** is an enterprise identity federation bridge inspired by Ory Polis. It connects enterprise Identity Providers (Okta, Microsoft Entra ID / Azure AD, Google Workspace, PingIdentity) to the Autorix IAM suite via **SAML 2.0 Web SSO** and **SCIM 2.0 (RFC 7643 / RFC 7644)** automated user provisioning.

---

## 🏛️ 1. Architecture & Protocol Topology

```text
   [ Enterprise IdP ] (Okta / Entra ID / Google)
           │                                 │
           ▼ (SAML 2.0 Web SSO)              ▼ (SCIM 2.0 Inbound Sync RFC 7644)
   ┌────────────────────────────────────────────────────────────────────────┐
   │                          Autorix Hermes                                │
   │                                                                        │
   │  ┌───────────────────────────┐  ┌───────────────────────────────────┐  │
   │  │ SAML 2.0 SP Engine        │  │ SCIM 2.0 Directory Sync Server    │  │
   │  │ (AuthnRequest, ACS, XML)  │  │ (/Users, /Groups, /Schemas)       │  │
   │  └─────────────┬─────────────┘  └─────────────────┬─────────────────┘  │
   └────────────────┼──────────────────────────────────┼────────────────────┘
                    │                                  │
                    ▼                                  ▼
           [ Autorix Ego ] ◄───────────────────────────┘
           (Synchronizes users & groups into Ego Identity database)
```

---

## 🏢 2. SAML 2.0 Single Sign-On (SSO)

Hermes acts as a SAML 2.0 **Service Provider (SP)**. It generates signed `AuthnRequest` XML messages and parses incoming assertions from IdPs at its Assertion Consumer Service (ACS).

### 2.1 Service Provider Metadata (`GET /saml/metadata`)

Enterprise identity administrators download Hermes SP metadata for import into Okta, Azure AD, or Google Workspace:

```bash
curl -s http://localhost:4477/saml/metadata
```

---

## 👥 3. SCIM 2.0 Directory Synchronization (RFC 7643 / RFC 7644)

SCIM enables continuous, automated user and group provisioning and deprovisioning when employees join, change roles, or leave the company.

---

## 📡 4. Complete REST & XML API Reference

Hermes operates on port `4477`.

### 4.1 SAML 2.0 Endpoints

- `GET /saml/metadata`: Returns SP metadata XML envelope.
- `GET /saml/login?provider={id}`: Generates an `AuthnRequest` URL and redirects the browser to the enterprise IdP.
- `POST /saml/acs`: Assertion Consumer Service endpoint handling incoming `SAMLResponse` form submissions.

---

### 4.2 SAML IdP Provider Administration

- `POST /admin/saml/providers`: Register a new SAML IdP provider.
- `GET /admin/saml/providers`: List registered SAML providers with pagination.
- `GET /admin/saml/providers/{id}`: Get provider configuration.
- `PATCH /admin/saml/providers/{id}`: Update provider settings or attribute mappings.
- `DELETE /admin/saml/providers/{id}`: Remove a provider.
- `POST /admin/saml/providers/{id}/enable`: Enable provider.
- `POST /admin/saml/providers/{id}/disable`: Disable provider.

#### Register SAML Provider Request Body
```json
{
  "id": "idp_okta_enterprise",
  "display_name": "Acme Corp Okta",
  "idp_entity_id": "http://www.okta.com/exk123456789",
  "idp_sso_url": "https://acme.okta.com/app/acme_autorix/sso/saml",
  "idp_certificate_pem": "-----BEGIN CERTIFICATE-----\nMIIDpDCCAoygAwIBAgIGAX...\n-----END CERTIFICATE-----",
  "attribute_mapping": {
    "email": "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress",
    "name": "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name",
    "department": "department"
  },
  "enabled": true
}
```

---

### 4.3 SCIM 2.0 Endpoints

- `GET /scim/v2/ServiceProviderConfig`: Returns SCIM capabilities (filter, patch, authentication schemes).
- `GET /scim/v2/Users`: Lists SCIM users formatted according to RFC 7643.
- `POST /scim/v2/Users`: Provisions a new user identity.
- `GET /scim/v2/Groups`: Lists SCIM groups.
- `POST /scim/v2/Groups`: Provisions a new group.
- `GET /scim/v2/Groups/{id}`: Retrieves a group by UUID.
- `PUT /scim/v2/Groups/{id}` & `PATCH /scim/v2/Groups/{id}`: Updates group members and display name.
- `DELETE /scim/v2/Groups/{id}`: Deletes a group.

---

### 4.4 SCIM Sync History & Auditing

- `GET /admin/scim/sync-history`: Retrieves execution logs of past directory sync webhooks.
- `POST /admin/scim/sync-history`: Records a synchronization audit batch.

---

## 🛠️ 5. Production Recipes

### Okta SAML 2.0 Integration Recipe
1. In Okta Admin Console, create a new SAML 2.0 Application.
2. Set Single Sign-On URL to `http://<hermes-host>:4477/saml/acs`.
3. Set Audience URI (SP Entity ID) to `http://<hermes-host>:4477/saml/metadata`.
4. Copy Okta's Identity Provider Single Sign-On URL and X.509 Certificate.
5. Register provider in Hermes via `POST /admin/saml/providers`.
