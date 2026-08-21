# Federate identities via SAML 2.0 and SCIM 2.0 with Autorix Hermes

Autorix Hermes is an enterprise identity federation bridge that connects enterprise Identity Providers (Okta, Microsoft Entra ID / Azure AD, Google Workspace, PingIdentity) to the Autorix IAM suite via SAML 2.0 Web SSO and SCIM 2.0 automated user provisioning.

### Quick path

1. Ensure Hermes is running on port `4477`.
2. Retrieve the Service Provider Metadata XML from `GET /saml/metadata`.
3. In your enterprise IdP (e.g., Okta), create a new SAML 2.0 application using the metadata.
4. Set the IdP's SSO URL and Certificate, and register the provider in Hermes via `POST /admin/saml/providers`.
5. Initiate a login using `GET /saml/login?provider={id}`.

### Details

#### Architecture
Hermes acts as a SAML 2.0 Service Provider (SP), generating signed `AuthnRequest` XML messages and parsing incoming assertions at its Assertion Consumer Service (ACS). It syncs users and groups directly into the Autorix Ego Identity database.

#### SAML 2.0 Single Sign-On APIs
* **Metadata**: `GET /saml/metadata` to download Hermes SP metadata for import into external IdPs.
* **Login Initiation**: `GET /saml/login?provider={id}` generates an `AuthnRequest` URL and redirects to the IdP.
* **ACS**: `POST /saml/acs` handles incoming `SAMLResponse` form submissions.

#### Provider Administration APIs
* **Register & Manage Providers**: Use `POST /admin/saml/providers` to register a new SAML IdP. You can update (`PATCH`), list (`GET`), enable/disable (`POST`), or delete (`DELETE`) them.

#### SCIM 2.0 Directory Synchronization
Hermes supports SCIM 2.0 (RFC 7643 / RFC 7644) for continuous, automated user and group provisioning and deprovisioning.
* **SCIM Endpoints**: `GET /scim/v2/ServiceProviderConfig`, `/Users`, and `/Groups` (with POST, PUT, PATCH, and DELETE operations) enable syncing identities.
* **Sync History**: `GET /admin/scim/sync-history` retrieves logs of past directory sync webhooks.

### Checklist

- [ ] Retrieve Hermes SP metadata (`/saml/metadata`).
- [ ] Configure the IdP (Okta, Entra ID) with the ACS URL and Entity ID.
- [ ] Register the IdP provider in Hermes with the correct certificate and attribute mappings.
- [ ] Configure SCIM 2.0 provisioning in the IdP for automated user synchronization.

### Next step

- Review the [Console usage guide](./console_usage_guide.md) to manage federated users through the Enterprise Studio.
- Learn about the underlying identity store in the [Ego usage guide](./ego_usage_guide.md).
