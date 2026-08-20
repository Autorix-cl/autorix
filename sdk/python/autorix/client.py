import typing
from dataclasses import dataclass, field
import httpx

@dataclass
class User:
    id: str
    email: typing.Optional[str] = None
    roles: typing.List[str] = field(default_factory=list)
    traits: typing.Dict[str, typing.Any] = field(default_factory=dict)

@dataclass
class PermissionCheckResult:
    allowed: bool
    reason: typing.Optional[str] = None

class AutorixClient:
    def __init__(
        self,
        ego_url: str = "http://localhost:4433",
        nexus_url: str = "http://localhost:8080",
        janus_url: str = "http://localhost:4444",
        vulcan_url: str = "http://localhost:4466",
        enable_cache: bool = True,
    ):
        self.ego_url = ego_url.rstrip("/")
        self.nexus_url = nexus_url.rstrip("/")
        self.janus_url = janus_url.rstrip("/")
        self.vulcan_url = vulcan_url.rstrip("/")
        self.enable_cache = enable_cache
        self._cache: typing.Dict[str, bool] = {}

    def check(
        self,
        namespace: str,
        object_id: str,
        relation: str,
        subject: str,
        context: typing.Optional[typing.Dict[str, typing.Any]] = None,
    ) -> PermissionCheckResult:
        cache_key = f"{namespace}:{object_id}#{relation}@{subject}"
        if self.enable_cache and cache_key in self._cache:
            return PermissionCheckResult(allowed=self._cache[cache_key], reason="cached")

        try:
            payload = {
                "namespace": namespace,
                "object": object_id,
                "relation": relation,
                "subject": subject,
                "context": context or {},
            }
            with httpx.Client(timeout=5.0) as client:
                res = client.post(f"{self.nexus_url}/check", json=payload)
                if res.status_code == 200:
                    data = res.json()
                    allowed = bool(data.get("allowed", False))
                    reason = data.get("reason", "nexus evaluation")
                else:
                    # Fail-closed on non-200 status
                    allowed = False
                    reason = f"Nexus returned status {res.status_code}"
        except Exception as e:
            # Fail-closed on network/connection error
            allowed = False
            reason = f"Nexus check error: {str(e)}"

        if self.enable_cache:
            self._cache[cache_key] = allowed

        return PermissionCheckResult(allowed=allowed, reason=reason)

    async def acheck(
        self,
        namespace: str,
        object_id: str,
        relation: str,
        subject: str,
        context: typing.Optional[typing.Dict[str, typing.Any]] = None,
    ) -> PermissionCheckResult:
        cache_key = f"{namespace}:{object_id}#{relation}@{subject}"
        if self.enable_cache and cache_key in self._cache:
            return PermissionCheckResult(allowed=self._cache[cache_key], reason="cached")

        try:
            payload = {
                "namespace": namespace,
                "object": object_id,
                "relation": relation,
                "subject": subject,
                "context": context or {},
            }
            async with httpx.AsyncClient(timeout=5.0) as client:
                res = await client.post(f"{self.nexus_url}/check", json=payload)
                if res.status_code == 200:
                    data = res.json()
                    allowed = bool(data.get("allowed", False))
                    reason = data.get("reason", "nexus evaluation")
                else:
                    allowed = False
                    reason = f"Nexus returned status {res.status_code}"
        except Exception as e:
            allowed = False
            reason = f"Nexus check error: {str(e)}"

        if self.enable_cache:
            self._cache[cache_key] = allowed

        return PermissionCheckResult(allowed=allowed, reason=reason)
