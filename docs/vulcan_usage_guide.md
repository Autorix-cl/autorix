# Issue and Attenuate API Keys with Autorix Vulcan

Autorix Vulcan issues high-entropy, environment-prefixed API keys that support offline, client-side caveat attenuation using HMAC-SHA256 cryptographic chaining. 

## Quick path

Vulcan runs its REST API on port `4466`.

1. **Create an API Key** (`POST /keys`):
   ```json
   {
     "name": "ETL Ingestion Pipeline",
     "owner_id": "svc_etl_runner",
     "scopes": ["ingest:write", "metrics:read"],
     "is_live": true
   }
   ```

2. **Attenuate the Key Offline**:
   Using the Autorix SDK, a primary service can restrict ("attenuate") a root Macaroon by appending a caveat (e.g., `time_before = NOW + 10m`) before passing it to a worker, **without making any database calls to Vulcan**.

3. **Verify the Macaroon** (`POST /keys/verify`):
   ```json
   {
     "macaroon": { "location": "http://localhost:4466", "key_id": "...", "caveats": [...], "signature": "..." },
     "context": { "now": "2026-08-20T14:30:00Z" }
   }
   ```

## Details

### Core Architecture & Macaroon Cryptography

Instead of plain API keys, Vulcan uses Macaroons. Each caveat appended updates the HMAC signature (`Sig_i+1 = HMAC-SHA256(Sig_i, Caveat_i+1)`). An attacker cannot remove or alter a caveat without invalidating the cryptographic chain.

```text
  [ Master Client / SDK ] ──(Attenuates token offline)──► [ Distributed Worker / Lambda ]
                                                                 │
                                                                 ▼ (Calls API)
  ┌────────────────────────────────────────────────────────────────────────┐
  │                            Autorix Vulcan                              │
  │                                                                        │
  │  ┌──────────────────────┐  ┌────────────────────────────────────────┐  │
  │  │ Prefixed Key Minter  │  │ HMAC-SHA256 Chained Macaroon Verifier  │  │
  │  │ (av_live_ / av_test_)│  │ (Evaluates caveats & verifies sig)     │  │
  │  └──────────┬───────────┘  └───────────────────┬────────────────────┘  │
  └─────────────┼──────────────────────────────────┼───────────────────────┘
                │                                  │
                ▼                                  ▼
            [ PostgreSQL Database: api_keys, scopes, usage_metrics ]
```

### Environment Prefixes & Key Storage

Vulcan uses environment prefixes to help secrets scanners (like GitHub Secret Scanning) detect leaked credentials:
* `av_live_`: Production access with real data.
* `av_test_`: Staging / Sandbox access for testing.

**Storage Security at Rest:**
Vulcan never stores plaintext keys. It stores `key_hash = SHA-256(raw_key)` and a `key_hint`. Constant-time comparison is used to eliminate timing attacks.

### Complete REST API Reference

* **`POST /keys`**: Create a new API key.
* **`GET /keys`**: List metadata of all issued API keys (cursor paginated).
* **`POST /keys/attenuate`**: Attenuate a Macaroon by appending a first-party caveat and returning the updated token.
* **`POST /keys/verify`**: Evaluate the signature chain and test all caveat assertions against the provided runtime context.
* **`POST /admin/keys/{id}/rotate`**: Generate a new root signature key while preserving the previous key during a grace period.
* **`DELETE /keys/{id}`**: Immediately revoke an API key.
* **`GET /admin/scopes` & `POST /admin/scopes`**: Manage system scopes.

### Production Recipes: Offline Worker Delegation

1. Primary service receives root Macaroon `M0`.
2. Primary service attenuates `M0` with a caveat (e.g., `time_before = NOW + 10m` and `method = POST`) to create `M1` locally in memory using the Autorix SDK.
3. Primary service passes `M1` to the worker thread.
4. Worker calls the API using `M1`. Vulcan verifies `M1` without any database lookup to fetch extra permissions.

## Checklist

* [ ] Select the correct prefix (`av_live_` vs `av_test_`) for your environment.
* [ ] Avoid storing raw keys in your database; always hash them securely.
* [ ] Configure your services to attenuate keys down to the minimum necessary privileges before delegating them to workers.
* [ ] Verify that your caveat assertions match the context provided during `verify`.

## Next step

Provision your first `av_test_` key via the `/keys` endpoint and practice attenuating it offline before integrating it into a production worker pipeline.
