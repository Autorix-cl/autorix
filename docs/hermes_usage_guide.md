# Autorix Hermes: Enterprise SAML 2.0 & SCIM 2.0 Federation Manual

**Autorix Hermes** is an enterprise identity federation bridge inspired by Ory Polis. It bridges enterprise Identity Providers (Okta, Microsoft Entra ID / Azure AD, Google Workspace, PingIdentity) to the Autorix IAM suite via **SAML 2.0** and **SCIM 2.0 (RFC 7643 / RFC 7644)** automated user provisioning.

---

## 🏛️ 1. Architecture & Protocol Topology

```text
   [ Enterprise IdP ] (Okta / Entra ID / Google)
           │                                 │
           ▼ (SAML 2.0 Web SSO)              ▼ (SCIM 2.0 Continuous Sync RFC 7644)
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
           (Creates & syncs identities and groups in Ego)
```

---

## 🏢 2. SAML 2.0 Single Sign-On (SSO)

Hermes acts as a SAML 2.0 **Service Provider (SP)**. It generates signed `AuthnRequest` XML envelopes and parses incoming assertions from IdPs at its Assertion Consumer Service (ACS).

### 2.1 Service Provider (SP) Metadata

Enterprise administrators can download Hermes SP metadata for import into Okta/Azure:

```bash
curl -s http://localhost:4477/saml/metadata
```

**XML Response:**
```xml
<EntityDescriptor entityID="http://localhost:4477/saml/metadata" xmlns="urn:oasis:names:tc:SAML:2.0:metadata">
  <SPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
    <NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress</NameIDFormat>
    <AssertionConsumerService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"
                              Location="http://localhost:4477/saml/acs"
                              index="1" isDefault="true"/>
  </SPSSODescriptor>
</EntityDescriptor>
```

### 2.2 Registering an Enterprise SAML IdP

```bash
curl -X POST http://localhost:4477/admin/saml/providers \
  -H "Content-Type: application/json" \
  -d '{
    "id": "idp_okta_enterprise",
    "name": "Acme Corp Okta",
    "entity_id": "http://www.okta.com/exk123456789",
    "sso_url": "https://acme.okta.com/app/acme_autorix/sso/saml",
    "x509_cert": "-----BEGIN CERTIFICATE-----\nMIIDpDCCAoygAwIBAgIGAX...\n-----END CERTIFICATE-----",
    "attribute_mapping": {
      "email": "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress",
      "name": "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name",
      "department": "department"
    }
  }'
```

---

## 👥 3. SCIM 2.0 User & Group Synchronization (RFC 7643 / 7644)

SCIM (System for Cross-domain Identity Management) enables automated identity provisioning and deprovisioning when employees join, change roles, or leave an enterprise.

### 3.1 Service Provider Configuration (`GET /scim/v2/ServiceProviderConfig`)

```bash
curl -s http://localhost:4477/scim/v2/ServiceProviderConfig
```

**Response:**
```json
{
  "schemas": ["urn:ietf:params:scim:schemas:core:2.0:ServiceProviderConfig"],
  "patch": { "supported": true },
  "bulk": { "supported": false },
  "filter": { "supported": true, "maxResults": 100 },
  "changePassword": { "supported": false },
  "sort": { "supported": false },
  "etag": { "supported": false },
  "authenticationSchemes": [
    {
      "name": "OAuth Bearer Token",
      "description": "Authentication scheme using the OAuth Bearer Standard",
      "specUri": "http://www.rfc-editor.org/info/rfc6750",
      "type": "oauthbearertoken",
      "primary": true
    }
  ]
}
```

### 3.2 Provisioning a User via SCIM (`POST /scim/v2/Users`)

```bash
curl -X POST http://localhost:4477/scim/v2/Users \
  -H "Content-Type: application/scim+json" \
  -H "Authorization: Bearer <scim_api_token>" \
  -d '{
    "schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"],
    "userName": "john.wick@continental.com",
    "name": {
      "givenName": "John",
      "familyName": "Wick"
    },
    "emails": [
      {
        "value": "john.wick@continental.com",
        "primary": true
      }
    ],
    "active": true
  }'
```

**Response (`201 Created`):**
```json
{
  "schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"],
  "id": "usr_scim_998877",
  "userName": "john.wick@continental.com",
  "name": { "givenName": "John", "familyName": "Wick" },
  "active": true,
  "meta": {
    "resourceType": "User",
    "created": "2026-08-20T09:00:00Z",
    "location": "http://localhost:4477/scim/v2/Users/usr_scim_998877"
  }
}
```

### 3.3 Deactivating a Deprovisioned User (`PATCH /scim/v2/Users/{id}`)

```bash
curl -X PATCH http://localhost:4477/scim/v2/Users/usr_scim_998877 \
  -H "Content-Type: application/scim+json" \
  -H "Authorization: Bearer <scim_api_token>" \
  -d '{
    "schemas": ["urn:ietf:params:scim:api:messages:2.0:PatchOp"],
    "Operations": [
      {
        "op": "replace",
        "path": "active",
        "value": false
      }
    ]
  }'
```

Hermes immediately updates the identity state in Ego to `suspended` and revokes all active sessions.

### 3.4 Group Management (`/scim/v2/Groups`)

* `GET /scim/v2/Groups`: Lists synchronized directory groups.
* `POST /scim/v2/Groups`: Creates a group with initial members.
* `PATCH /scim/v2/Groups/{id}`: Adds or removes group members.
