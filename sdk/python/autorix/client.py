"""Typed client for Autorix public runtime APIs.

The SDK deliberately exposes only public runtime endpoints. Administrative APIs
remain service-to-service concerns and are not reachable through this client.
"""
from __future__ import annotations

import asyncio
import random
import time
from dataclasses import dataclass, field
from typing import Any, Dict, Iterable, List, Mapping, Optional, Sequence, Tuple
from urllib.parse import urlencode

import httpx


class AutorixError(Exception):
    """Base error raised by the Autorix SDK."""


class AutorixHTTPError(AutorixError):
    def __init__(self, status_code: int, message: str, payload: Any = None):
        super().__init__(message)
        self.status_code = status_code
        self.payload = payload


@dataclass
class User:
    id: str
    email: Optional[str] = None
    roles: List[str] = field(default_factory=list)
    traits: Dict[str, Any] = field(default_factory=dict)
    is_machine: bool = False


@dataclass
class Session:
    id: str
    identity_id: str
    expires_at: Optional[str] = None
    authenticated_at: Optional[str] = None
    identity: Optional[Dict[str, Any]] = None


@dataclass
class PermissionCheckResult:
    allowed: bool
    reason: Optional[str] = None
    trace: Optional[Dict[str, Any]] = None
    snap_token: Optional[str] = None


@dataclass
class PolicyEvaluationResult:
    all_passed: bool
    total_evaluated: int
    results: List[Dict[str, Any]] = field(default_factory=list)


@dataclass
class VerifyKeyResult:
    valid: bool
    api_key: Optional[Dict[str, Any]] = None
    error: Optional[str] = None
    # Compatibility fields retained for 1.0 users.
    key_id: Optional[str] = None
    name: Optional[str] = None
    scopes: List[str] = field(default_factory=list)


@dataclass
class Page:
    items: List[Dict[str, Any]]
    next_cursor: Optional[str] = None
    has_more: bool = False


@dataclass
class TokenResponse:
    access_token: str
    token_type: str
    expires_in: int
    refresh_token: Optional[str] = None
    id_token: Optional[str] = None
    scope: Optional[str] = None


class RetryConfig:
    def __init__(self, max_retries: int = 3, initial_delay: float = 0.05, max_delay: float = 2.0, backoff_factor: float = 2.0):
        self.max_retries = max_retries
        self.initial_delay = initial_delay
        self.max_delay = max_delay
        self.backoff_factor = backoff_factor


class AutorixClient:
    """Synchronous public-runtime client.

    `http_client` is injectable for tests. It is never closed by the SDK when
    supplied by the caller.
    """
    def __init__(self, base_url: str = "http://localhost:4455", ego_url: str = "http://localhost:4433", nexus_url: str = "http://localhost:8080", themis_url: str = "http://localhost:4488", janus_url: str = "http://localhost:4444", vulcan_url: str = "http://localhost:4466", argus_url: str = "http://localhost:4400", api_key: Optional[str] = None, enable_cache: bool = True, cache_ttl: float = 10.0, retry_config: Optional[RetryConfig] = None, timeout: float = 5.0, http_client: Optional[httpx.Client] = None):
        self.base_url, self.ego_url, self.nexus_url = base_url.rstrip("/"), ego_url.rstrip("/"), nexus_url.rstrip("/")
        self.themis_url, self.janus_url, self.vulcan_url, self.argus_url = themis_url.rstrip("/"), janus_url.rstrip("/"), vulcan_url.rstrip("/"), argus_url.rstrip("/")
        self.api_key, self.enable_cache, self.cache_ttl = api_key, enable_cache, cache_ttl
        self.retry_config, self.timeout = retry_config or RetryConfig(), timeout
        self._cache: Dict[str, Tuple[bool, float]] = {}
        self._http = http_client or httpx.Client(timeout=timeout, follow_redirects=False)
        self._owns_http = http_client is None

    def close(self) -> None:
        if self._owns_http:
            self._http.close()

    def __enter__(self) -> "AutorixClient": return self
    def __exit__(self, *_: Any) -> None: self.close()

    def _headers(self, headers: Optional[Mapping[str, str]] = None) -> Dict[str, str]:
        result = {"User-Agent": "Autorix-Python-SDK/1.0.0", "Accept": "application/json"}
        if self.api_key: result["Authorization"] = f"Bearer {self.api_key}"
        if headers: result.update(headers)
        return result

    def _delay(self, attempt: int) -> float:
        return random.uniform(0, min(self.retry_config.initial_delay * self.retry_config.backoff_factor ** (attempt - 1), self.retry_config.max_delay))

    def _request(self, method: str, url: str, *, json: Any = None, data: Any = None, params: Optional[Mapping[str, Any]] = None, headers: Optional[Mapping[str, str]] = None, retry: bool = False) -> Any:
        # Only idempotent reads explicitly opt into retries. Writes and OAuth
        # token endpoints must never be replayed by the SDK.
        attempts = self.retry_config.max_retries if retry and method.upper() in {"GET", "HEAD"} else 0
        for attempt in range(attempts + 1):
            try:
                response = self._http.request(method, url, json=json, data=data, params=params, headers=self._headers(headers), timeout=self.timeout)
                if response.status_code in {429, 502, 503, 504} and attempt < attempts:
                    time.sleep(self._delay(attempt + 1)); continue
                if response.is_error:
                    try: payload = response.json()
                    except ValueError: payload = response.text
                    message = payload.get("error", payload.get("message", response.reason_phrase)) if isinstance(payload, dict) else str(payload)
                    raise AutorixHTTPError(response.status_code, message, payload)
                if response.status_code == 204 or not response.content: return None
                return response.json()
            except httpx.HTTPError as exc:
                if attempt == attempts: raise AutorixError(str(exc)) from exc
                time.sleep(self._delay(attempt + 1))
        raise AssertionError("unreachable")

    # Nexus
    def check(self, namespace: str, object_id: str, relation: str, subject: str, subject_namespace: str = "user", context: Optional[Dict[str, Any]] = None, explain: bool = False, subject_relation: str = "", snap_token: Optional[str] = None) -> PermissionCheckResult:
        key = f"{namespace}:{object_id}#{relation}@{subject_namespace}:{subject}:{subject_relation}"
        if self.enable_cache and key in self._cache and time.time() < self._cache[key][1]: return PermissionCheckResult(self._cache[key][0], "cached")
        payload = {"namespace": namespace, "object": object_id, "relation": relation, "subject_id": subject, "subject_namespace": subject_namespace, "subject_relation": subject_relation, "request_context": context or {}, "explain": explain}
        if snap_token: payload["snap_token"] = snap_token
        data = self._request("POST", f"{self.nexus_url}/check", json=payload)
        result = PermissionCheckResult(bool(data.get("allowed")), data.get("reason"), data.get("trace"), data.get("snap_token") or data.get("zookie"))
        if self.enable_cache: self._cache[key] = (result.allowed, time.time() + self.cache_ttl)
        return result

    async def acheck(self, *args: Any, **kwargs: Any) -> PermissionCheckResult:
        return await asyncio.to_thread(self.check, *args, **kwargs)

    def check_batch(self, requests: List[Dict[str, Any]]) -> List[PermissionCheckResult]:
        return [self.check(namespace=r["namespace"], object_id=r["object_id"], relation=r["relation"], subject=r["subject"], subject_namespace=r.get("subject_namespace", "user"), context=r.get("context"), explain=r.get("explain", False), subject_relation=r.get("subject_relation", ""), snap_token=r.get("snap_token")) for r in requests]

    def expand(self, namespace: str, object_id: str, relation: str) -> Dict[str, Any]:
        return self._request("POST", f"{self.nexus_url}/expand", json={"namespace": namespace, "object": object_id, "relation": relation})["tree"]

    def lookup_subjects(self, namespace: str, object_id: str, relation: str) -> List[Dict[str, Any]]:
        return self._request("POST", f"{self.nexus_url}/lookup/subjects", json={"namespace": namespace, "object": object_id, "relation": relation}).get("subjects", [])

    def lookup_resources(self, namespace: str, relation: str, subject: str, subject_namespace: str = "user", subject_relation: str = "") -> List[str]:
        data = self._request("POST", f"{self.nexus_url}/lookup/resources", json={"namespace": namespace, "relation": relation, "subject_id": subject, "subject_namespace": subject_namespace, "subject_relation": subject_relation})
        return data.get("resources", [])

    def list_tuples(self, namespace: Optional[str] = None, cursor: Optional[str] = None, limit: Optional[int] = None) -> Page:
        data = self._request("GET", f"{self.nexus_url}/tuples", params={k: v for k, v in {"namespace": namespace, "cursor": cursor, "limit": limit}.items() if v is not None}, retry=True)
        return Page(data.get("items", data.get("data", [])), data.get("next_cursor"), bool(data.get("has_more")))

    def write_tuples(self, tuples: Sequence[Mapping[str, Any]]) -> List[Dict[str, Any]]:
        return self._request("POST", f"{self.nexus_url}/tuples", json=list(tuples))

    def delete_tuples(self, tuples: Sequence[Mapping[str, Any]]) -> None:
        self._request("DELETE", f"{self.nexus_url}/tuples", json=list(tuples))

    # Themis
    def evaluate_policy(self, policy_context: Dict[str, Any], tenant_id: str = "default", policy_id: str = "", label_filter: Optional[Dict[str, str]] = None) -> PolicyEvaluationResult:
        data = self._request("POST", f"{self.themis_url}/policies/evaluate", json={"tenant_id": tenant_id, "policy_id": policy_id, "payload": policy_context, "label_filter": label_filter or {}})
        return PolicyEvaluationResult(bool(data.get("all_passed", data.get("AllPassed", False))), int(data.get("total_evaluated", data.get("TotalEvaluated", 0))), data.get("results", data.get("Results", [])))

    def list_policies(self, tenant_id: str = "default", enabled_only: bool = False, cursor: Optional[str] = None, limit: Optional[int] = None) -> Page:
        data = self._request("GET", f"{self.themis_url}/policies", params={"tenant_id": tenant_id, "enabled_only": str(enabled_only).lower(), **{k:v for k,v in {"cursor":cursor,"limit":limit}.items() if v is not None}}, retry=True)
        return Page(data.get("items", data.get("data", [])), data.get("next_cursor"), bool(data.get("has_more")))

    def create_policy(self, policy: Mapping[str, Any]) -> Dict[str, Any]: return self._request("POST", f"{self.themis_url}/policies", json=dict(policy))
    def get_policy(self, policy_id: str, tenant_id: str = "default") -> Dict[str, Any]: return self._request("GET", f"{self.themis_url}/policies/{policy_id}", params={"tenant_id": tenant_id}, retry=True)
    def update_policy(self, policy_id: str, policy: Mapping[str, Any]) -> Dict[str, Any]: return self._request("PUT", f"{self.themis_url}/policies/{policy_id}", json=dict(policy))
    def delete_policy(self, policy_id: str, tenant_id: str = "default") -> None: self._request("DELETE", f"{self.themis_url}/policies/{policy_id}", params={"tenant_id": tenant_id})
    def validate_policy(self, expression: str) -> Dict[str, Any]: return self._request("POST", f"{self.themis_url}/policies/validate", json={"expression": expression})
    def dry_run_policy(self, expression: str, payload: Optional[Mapping[str, Any]] = None) -> Dict[str, Any]: return self._request("POST", f"{self.themis_url}/policies/dry-run", json={"expression": expression, "payload": dict(payload or {})})
    def list_policy_versions(self, policy_id: str, tenant_id: str = "default") -> List[Dict[str, Any]]: return self._request("GET", f"{self.themis_url}/policies/{policy_id}/versions", params={"tenant_id": tenant_id}, retry=True)
    def list_policy_fixtures(self, policy_id: str, tenant_id: str = "default") -> List[Dict[str, Any]]: return self._request("GET", f"{self.themis_url}/policies/{policy_id}/fixtures", params={"tenant_id": tenant_id}, retry=True)
    def create_policy_fixture(self, policy_id: str, name: str, expected_result: bool, payload: Optional[Mapping[str, Any]] = None, description: str = "", tenant_id: str = "default") -> Dict[str, Any]: return self._request("POST", f"{self.themis_url}/policies/{policy_id}/fixtures", params={"tenant_id": tenant_id}, json={"name": name, "expected_result": expected_result, "payload": dict(payload or {}), "description": description})
    def delete_policy_fixture(self, policy_id: str, fixture_id: str, tenant_id: str = "default") -> None: self._request("DELETE", f"{self.themis_url}/policies/{policy_id}/fixtures/{fixture_id}", params={"tenant_id": tenant_id})
    def run_policy_test_suite(self, policy_id: str, tenant_id: str = "default") -> Dict[str, Any]: return self._request("POST", f"{self.themis_url}/policies/{policy_id}/test-suite", params={"tenant_id": tenant_id})

    # Vulcan
    def create_api_key(self, name: str, owner_id: str, scopes: Sequence[str], description: str = "", expires_at: Optional[str] = None, is_live: bool = True) -> Dict[str, Any]:
        payload = {"name": name, "owner_id": owner_id, "scopes": list(scopes), "description": description, "is_live": is_live}
        if expires_at: payload["expires_at"] = expires_at
        return self._request("POST", f"{self.vulcan_url}/keys", json=payload)
    def list_api_keys(self, cursor: Optional[str] = None, limit: Optional[int] = None) -> Page:
        data = self._request("GET", f"{self.vulcan_url}/keys", params={k:v for k,v in {"cursor":cursor,"limit":limit}.items() if v is not None}, retry=True); return Page(data.get("items", data.get("data", [])), data.get("next_cursor"), bool(data.get("has_more")))
    def attenuate_api_key(self, macaroon: Mapping[str, Any], caveat: str) -> Dict[str, Any]: return self._request("POST", f"{self.vulcan_url}/keys/attenuate", json={"macaroon": dict(macaroon), "caveat": caveat})
    def verify_api_key(self, token: Any, eval_context: Optional[Dict[str, Any]] = None) -> VerifyKeyResult:
        # `token` was historically documented as a string. The server contract
        # has always required a macaroon, so retain the parameter but fail safely.
        if not isinstance(token, Mapping): return VerifyKeyResult(valid=False, error="Vulcan verification requires a macaroon object")
        try:
            data = self._request("POST", f"{self.vulcan_url}/keys/verify", json={"macaroon": dict(token), "context": eval_context or {}})
            key = data.get("api_key") or {}; return VerifyKeyResult(True, key, key.get("id"), key.get("name"), key.get("scopes", []))
        except AutorixError as exc: return VerifyKeyResult(valid=False, error=str(exc))
    def revoke_api_key(self, key_id: str) -> None: self._request("DELETE", f"{self.vulcan_url}/keys/{key_id}")

    # Ego public session APIs
    def register(self, traits: Mapping[str, Any], password: str, *, flow: Optional[str] = None, csrf_token: str = "", method: str = "password") -> Dict[str, Any]:
        return self._request("POST", f"{self.ego_url}/self-service/registration", params={"flow": flow} if flow else None, json={"traits": dict(traits), "password": password, "csrf_token": csrf_token, "method": method})
    def login(self, identifier: str, password: str) -> Dict[str, Any]: return self._request("POST", f"{self.ego_url}/self-service/login", json={"identifier": identifier, "password": password})
    def whoami(self, session_token: str) -> Session:
        data = self._request("GET", f"{self.ego_url}/sessions/whoami", headers={"Authorization": f"Bearer {session_token}"}, retry=True); return Session(data["id"], data["identity_id"], data.get("expires_at"), data.get("authenticated_at"), data.get("identity"))
    def logout(self, session_token: str) -> None: self._request("POST", f"{self.ego_url}/self-service/logout", headers={"Authorization": f"Bearer {session_token}"})

    # Janus OAuth/OIDC. Secret-bearing operations are explicit backend APIs.
    def oidc_discovery(self) -> Dict[str, Any]: return self._request("GET", f"{self.janus_url}/.well-known/openid-configuration", retry=True)
    def jwks(self) -> Dict[str, Any]: return self._request("GET", f"{self.janus_url}/.well-known/jwks.json", retry=True)
    def authorization_url(self, client_id: str, redirect_uri: str, scope: Sequence[str], state: str, code_challenge: str, code_challenge_method: str = "S256", **extra: str) -> str:
        params = {"response_type": "code", "client_id": client_id, "redirect_uri": redirect_uri, "scope": " ".join(scope), "state": state, "code_challenge": code_challenge, "code_challenge_method": code_challenge_method, **extra}; return f"{self.janus_url}/oauth2/auth?{urlencode(params)}"
    def exchange_code(self, client_id: str, code: str, redirect_uri: str, code_verifier: str, client_secret: Optional[str] = None) -> TokenResponse:
        return self._token({"grant_type":"authorization_code", "client_id":client_id, "code":code, "redirect_uri":redirect_uri, "code_verifier":code_verifier}, client_secret)
    def refresh_token(self, client_id: str, refresh_token: str, client_secret: Optional[str] = None) -> TokenResponse: return self._token({"grant_type":"refresh_token", "client_id":client_id, "refresh_token":refresh_token}, client_secret)
    def client_credentials(self, client_id: str, client_secret: str, resource: str, scopes: Sequence[str] = ()) -> TokenResponse: return self._token({"grant_type":"client_credentials", "client_id":client_id, "resource":resource, "scope":" ".join(scopes)}, client_secret)
    def _token(self, form: Dict[str, str], client_secret: Optional[str]) -> TokenResponse:
        headers = {"Content-Type":"application/x-www-form-urlencoded"}; auth = (form["client_id"], client_secret) if client_secret else None
        # httpx BasicAuth cannot be transported through generic helper headers.
        response = self._http.post(f"{self.janus_url}/oauth2/token", data=form, headers=self._headers(headers), auth=auth, timeout=self.timeout)
        if response.is_error: raise AutorixHTTPError(response.status_code, response.text)
        data = response.json(); return TokenResponse(data["access_token"], data.get("token_type", "Bearer"), int(data.get("expires_in", 0)), data.get("refresh_token"), data.get("id_token"), data.get("scope"))
    def introspect(self, token: str, client_id: str, client_secret: str) -> Dict[str, Any]: return self._oauth_form("/oauth2/introspect", {"token": token, "client_id": client_id}, client_id, client_secret)
    def revoke_token(self, token: str, client_id: str, client_secret: str, token_type_hint: str = "access_token") -> None: self._oauth_form("/oauth2/revoke", {"token":token,"token_type_hint":token_type_hint,"client_id":client_id}, client_id, client_secret)
    def _oauth_form(self, path: str, form: Dict[str, str], client_id: str, client_secret: str) -> Any:
        response = self._http.post(f"{self.janus_url}{path}", data=form, headers=self._headers({"Content-Type":"application/x-www-form-urlencoded"}), auth=(client_id, client_secret), timeout=self.timeout)
        if response.is_error: raise AutorixHTTPError(response.status_code, response.text)
        return response.json() if response.content else None
