import asyncio
import random
import time
import typing
from dataclasses import dataclass, field
import httpx

@dataclass
class User:
    id: str
    email: typing.Optional[str] = None
    roles: typing.List[str] = field(default_factory=list)
    traits: typing.Dict[str, typing.Any] = field(default_factory=dict)
    is_machine: bool = False

@dataclass
class PermissionCheckResult:
    allowed: bool
    reason: typing.Optional[str] = None
    trace: typing.Optional[typing.Dict[str, typing.Any]] = None

@dataclass
class PolicyEvaluationResult:
    all_passed: bool
    total_evaluated: int
    results: typing.List[typing.Dict[str, typing.Any]] = field(default_factory=list)

@dataclass
class VerifyKeyResult:
    valid: bool
    key_id: typing.Optional[str] = None
    name: typing.Optional[str] = None
    scopes: typing.List[str] = field(default_factory=list)
    error: typing.Optional[str] = None

class RetryConfig:
    def __init__(
        self,
        max_retries: int = 3,
        initial_delay: float = 0.05,
        max_delay: float = 2.0,
        backoff_factor: float = 2.0,
    ):
        self.max_retries = max_retries
        self.initial_delay = initial_delay
        self.max_delay = max_delay
        self.backoff_factor = backoff_factor

class AutorixClient:
    def __init__(
        self,
        base_url: str = "http://localhost:4455",
        ego_url: str = "http://localhost:4433",
        nexus_url: str = "http://localhost:8080",
        themis_url: str = "http://localhost:4488",
        janus_url: str = "http://localhost:4444",
        vulcan_url: str = "http://localhost:4466",
        argus_url: str = "http://localhost:4400",
        api_key: typing.Optional[str] = None,
        enable_cache: bool = True,
        cache_ttl: float = 10.0,
        retry_config: typing.Optional[RetryConfig] = None,
    ):
        self.base_url = base_url.rstrip("/")
        self.ego_url = ego_url.rstrip("/")
        self.nexus_url = nexus_url.rstrip("/")
        self.themis_url = themis_url.rstrip("/")
        self.janus_url = janus_url.rstrip("/")
        self.vulcan_url = vulcan_url.rstrip("/")
        self.argus_url = argus_url.rstrip("/")
        self.api_key = api_key
        self.enable_cache = enable_cache
        self.cache_ttl = cache_ttl
        self.retry_config = retry_config or RetryConfig()
        self._cache: typing.Dict[str, typing.Tuple[bool, float]] = {}

    def _get_headers(self) -> typing.Dict[str, str]:
        headers = {
            "User-Agent": "Autorix-Python-SDK/1.0.0",
            "Content-Type": "application/json",
        }
        if self.api_key:
            headers["Authorization"] = f"Bearer {self.api_key}"
        return headers

    def _calculate_jitter_delay(self, attempt: int) -> float:
        delay = min(
            self.retry_config.initial_delay * (self.retry_config.backoff_factor ** (attempt - 1)),
            self.retry_config.max_delay,
        )
        return random.uniform(0, delay)

    def check(
        self,
        namespace: str,
        object_id: str,
        relation: str,
        subject: str,
        subject_namespace: str = "user",
        context: typing.Optional[typing.Dict[str, typing.Any]] = None,
        explain: bool = False,
    ) -> PermissionCheckResult:
        cache_key = f"{namespace}:{object_id}#{relation}@{subject_namespace}:{subject}"
        now = time.time()

        if self.enable_cache and cache_key in self._cache:
            allowed, exp = self._cache[cache_key]
            if now < exp:
                return PermissionCheckResult(allowed=allowed, reason="cached")

        payload = {
            "namespace": namespace,
            "object": object_id,
            "relation": relation,
            "subject_id": subject,
            "subject_namespace": subject_namespace,
            "request_context": context or {},
            "explain": explain,
        }

        allowed = False
        reason = "unresolved"
        trace = None

        for attempt in range(self.retry_config.max_retries + 1):
            if attempt > 0:
                time.sleep(self._calculate_jitter_delay(attempt))

            try:
                with httpx.Client(timeout=5.0) as client:
                    res = client.post(
                        f"{self.nexus_url}/check",
                        json=payload,
                        headers=self._get_headers(),
                    )
                    if res.status_code == 200:
                        data = res.json()
                        allowed = bool(data.get("allowed", False))
                        reason = data.get("reason", "granted" if allowed else "denied")
                        trace = data.get("trace")
                        break
                    elif res.status_code in (429, 502, 503, 504) and attempt < self.retry_config.max_retries:
                        continue
                    else:
                        allowed = False
                        reason = f"Nexus returned status {res.status_code}"
                        break
            except Exception as e:
                if attempt == self.retry_config.max_retries:
                    allowed = False
                    reason = f"Nexus check error: {str(e)}"
                    break

        if self.enable_cache:
            self._cache[cache_key] = (allowed, now + self.cache_ttl)

        return PermissionCheckResult(allowed=allowed, reason=reason, trace=trace)

    async def acheck(
        self,
        namespace: str,
        object_id: str,
        relation: str,
        subject: str,
        subject_namespace: str = "user",
        context: typing.Optional[typing.Dict[str, typing.Any]] = None,
        explain: bool = False,
    ) -> PermissionCheckResult:
        cache_key = f"{namespace}:{object_id}#{relation}@{subject_namespace}:{subject}"
        now = time.time()

        if self.enable_cache and cache_key in self._cache:
            allowed, exp = self._cache[cache_key]
            if now < exp:
                return PermissionCheckResult(allowed=allowed, reason="cached")

        payload = {
            "namespace": namespace,
            "object": object_id,
            "relation": relation,
            "subject_id": subject,
            "subject_namespace": subject_namespace,
            "request_context": context or {},
            "explain": explain,
        }

        allowed = False
        reason = "unresolved"
        trace = None

        for attempt in range(self.retry_config.max_retries + 1):
            if attempt > 0:
                await asyncio.sleep(self._calculate_jitter_delay(attempt))

            try:
                async with httpx.AsyncClient(timeout=5.0) as client:
                    res = await client.post(
                        f"{self.nexus_url}/check",
                        json=payload,
                        headers=self._get_headers(),
                    )
                    if res.status_code == 200:
                        data = res.json()
                        allowed = bool(data.get("allowed", False))
                        reason = data.get("reason", "granted" if allowed else "denied")
                        trace = data.get("trace")
                        break
                    elif res.status_code in (429, 502, 503, 504) and attempt < self.retry_config.max_retries:
                        continue
                    else:
                        allowed = False
                        reason = f"Nexus returned status {res.status_code}"
                        break
            except Exception as e:
                if attempt == self.retry_config.max_retries:
                    allowed = False
                    reason = f"Nexus check error: {str(e)}"
                    break

        if self.enable_cache:
            self._cache[cache_key] = (allowed, now + self.cache_ttl)

        return PermissionCheckResult(allowed=allowed, reason=reason, trace=trace)

    def check_batch(
        self,
        requests: typing.List[typing.Dict[str, typing.Any]],
    ) -> typing.List[PermissionCheckResult]:
        return [
            self.check(
                namespace=r["namespace"],
                object_id=r["object_id"],
                relation=r["relation"],
                subject=r["subject"],
                subject_namespace=r.get("subject_namespace", "user"),
                context=r.get("context"),
            )
            for r in requests
        ]

    def evaluate_policy(
        self,
        policy_context: typing.Dict[str, typing.Any],
        tenant_id: str = "default",
    ) -> PolicyEvaluationResult:
        try:
            with httpx.Client(timeout=5.0) as client:
                res = client.post(
                    f"{self.themis_url}/v1/policies/evaluate",
                    json={"tenant_id": tenant_id, "context": policy_context},
                    headers=self._get_headers(),
                )
                if res.status_code == 200:
                    data = res.json()
                    return PolicyEvaluationResult(
                        all_passed=bool(data.get("AllPassed", data.get("all_passed", False))),
                        total_evaluated=int(data.get("TotalEvaluated", data.get("total_evaluated", 0))),
                        results=data.get("Results", data.get("results", [])),
                    )
        except Exception:
            pass
        return PolicyEvaluationResult(all_passed=False, total_evaluated=0)

    def verify_api_key(
        self,
        token: str,
        eval_context: typing.Optional[typing.Dict[str, typing.Any]] = None,
    ) -> VerifyKeyResult:
        try:
            with httpx.Client(timeout=5.0) as client:
                res = client.post(
                    f"{self.vulcan_url}/keys/verify",
                    json={"token": token, "context": eval_context or {}},
                    headers=self._get_headers(),
                )
                if res.status_code == 200:
                    data = res.json()
                    return VerifyKeyResult(
                        valid=bool(data.get("valid", False)),
                        key_id=data.get("key_id"),
                        name=data.get("name"),
                        scopes=data.get("scopes", []),
                    )
                return VerifyKeyResult(valid=False, error=f"Status {res.status_code}")
        except Exception as e:
            return VerifyKeyResult(valid=False, error=str(e))
