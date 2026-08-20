# Python SDK Reference Manual (`autorix`)

The official **Autorix Python SDK** provides both synchronous and asynchronous (`asyncio` / `httpx`) client interfaces, FastAPI security dependencies, and Flask middleware for Python backends.

---

## 1. Installation

```bash
pip install autorix
```

Requirements: Python `3.9+`.

---

## 2. Client Initialization & Resilience

```python
from autorix import AutorixClient, RetryConfig

client = AutorixClient(
    base_url="http://localhost:4455",
    nexus_url="http://localhost:8080",
    themis_url="http://localhost:4488",
    ego_url="http://localhost:4433",
    vulcan_url="http://localhost:4466",
    api_key="av_live_01918a7b6c5d4e3f2a1b0c9d8e7f6a",
    
    # In-memory caching
    enable_cache=True,
    cache_ttl=15.0, # seconds
    
    # Resilience: Exponential backoff with full jitter
    retry_config=RetryConfig(
        max_retries=3,
        initial_delay=0.05, # 50ms
        max_delay=2.0,       # 2s
        backoff_factor=2.0
    )
)
```

---

## 3. Synchronous & Asynchronous Operations

### 3.1 Evaluating ReBAC Permissions (`check` & `acheck`)

#### Synchronous Check:
```python
result = client.check(
    namespace="documents",
    object_id="contract_2026_q3",
    relation="viewer",
    subject="usr_alice",
    context={"ip": "10.0.4.15"},
    explain=True
)

if result.allowed:
    print("Access granted via", result.reason)
```

#### Asynchronous Check (FastAPI / Starlette):
```python
result = await client.acheck(
    namespace="documents",
    object_id="contract_2026_q3",
    relation="editor",
    subject="usr_alice"
)
```

### 3.2 Vectorized Batch Checks (`check_batch`)

```python
requests = [
    {"namespace": "documents", "object_id": "doc_1", "relation": "read", "subject": "usr_alice"},
    {"namespace": "documents", "object_id": "doc_2", "relation": "write", "subject": "usr_alice"},
    {"namespace": "documents", "object_id": "doc_3", "relation": "admin", "subject": "usr_alice"},
]

batch_results = client.check_batch(requests)
for r in batch_results:
    print(f"Allowed: {r.allowed} ({r.reason})")
```

### 3.3 Themis ABAC Policy Evaluation (`evaluate_policy`)

```python
policy_result = client.evaluate_policy(
    tenant_id="default",
    policy_context={
        "request": {
            "auth": {"claims": {"department": "finance"}, "mfa": True},
            "time": {"hour": 14}
        },
        "resource": {"amount": 75000}
    }
)

if policy_result.all_passed:
    print(f"Policy evaluation passed ({policy_result.total_evaluated} policies checked)")
```

### 3.4 Validating API Keys & Macaroons (`verify_api_key`)

```python
res = client.verify_api_key(
    token="av_live_9f8e7d6c5b4a3f2e...",
    eval_context={"method": "POST", "required_scope": "ingest:write"}
)

if res.valid:
    print(f"Valid key for: {res.name} (Scopes: {res.scopes})")
```

---

## 🚀 4. Framework Integrations

### 4.1 FastAPI Integration

```python
from fastapi import FastAPI, Depends, Header, HTTPException
from autorix.fastapi import AutorixSecurity
from autorix import AutorixClient

app = FastAPI(title="Enterprise Orders API")
auth = AutorixSecurity()
autorix = AutorixClient(nexus_url="http://nexus:8080")

@app.get("/api/orders/{order_id}")
async def get_order(
    order_id: str,
    x_user_id: str = Header(None),
    x_user_email: str = Header(None)
):
    # 1. Extract identity from Aegis proxy headers
    user = auth.get_current_user(x_user_id=x_user_id, x_user_email=x_user_email)
    if not user:
        raise HTTPException(status_code=401, detail="Unauthorized")

    # 2. Evaluate ReBAC authorization
    res = await autorix.acheck("orders", order_id, "viewer", user.id)
    if not res.allowed:
        raise HTTPException(status_code=403, detail="Forbidden: insufficient permissions")

    return {
        "order_id": order_id,
        "customer": user.email,
        "status": "shipped"
    }
```

### 4.2 Flask Integration

```python
from flask import Flask, request, jsonify, g
from autorix import AutorixClient

app = Flask(__name__)
client = AutorixClient(nexus_url="http://localhost:8080")

@app.before_request
def extract_identity():
    user_id = request.headers.get("X-User-ID")
    if user_id:
        g.user_id = user_id
        g.user_email = request.headers.get("X-User-Email")

@app.route("/api/v1/vault/<vault_id>")
def access_vault(vault_id):
    if not hasattr(g, "user_id"):
        return jsonify({"error": "Unauthorized"}), 401

    allowed = client.check("vaults", vault_id, "accessor", g.user_id).allowed
    if not allowed:
        return jsonify({"error": "Forbidden"}), 403

    return jsonify({"vault_id": vault_id, "status": "unlocked"})
```
