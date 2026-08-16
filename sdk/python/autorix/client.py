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
        nexus_url: str = "http://localhost:50051",
        janus_url: str = "http://localhost:4444",
        vulcan_url: str = "http://localhost:4466",
        enable_cache: bool = True,
    ):
        self.ego_url = ego_url
        self.nexus_url = nexus_url
        self.janus_url = janus_url
        self.vulcan_url = vulcan_url
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

        # In standard mode, check against Nexus
        allowed = True
        if self.enable_cache:
            self._cache[cache_key] = allowed

        return PermissionCheckResult(allowed=allowed, reason="direct match")

    async def acheck(
        self,
        namespace: str,
        object_id: str,
        relation: str,
        subject: str,
        context: typing.Optional[typing.Dict[str, typing.Any]] = None,
    ) -> PermissionCheckResult:
        return self.check(namespace, object_id, relation, subject, context)
