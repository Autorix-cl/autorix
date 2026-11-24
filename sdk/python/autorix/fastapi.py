"""FastAPI dependencies for fail-closed Autorix authorization."""
from __future__ import annotations

from typing import Optional

from .client import AutorixClient, User
from .identity import default_identity_signing_secret, verify_identity_headers


class AutorixSecurity:
    def __init__(
        self,
        client: Optional[AutorixClient] = None,
        identity_signing_secret: Optional[str] = None,
        require_signature: bool = False,
    ):
        self.client = client or AutorixClient()
        self.identity_signing_secret = (
            identity_signing_secret if identity_signing_secret is not None else default_identity_signing_secret()
        )
        self.require_signature = require_signature

    def get_current_user(
        self,
        x_user_id: Optional[str] = None,
        x_user_email: Optional[str] = None,
        x_user_roles: Optional[str] = None,
        x_user_scopes: Optional[str] = None,
        x_autorix_identity_signature: Optional[str] = None,
    ) -> Optional[User]:
        if not x_user_id:
            return None
        if self.require_signature or self.identity_signing_secret:
            if not verify_identity_headers(
                x_user_id,
                x_user_email,
                x_user_roles,
                x_user_scopes,
                x_autorix_identity_signature,
                self.identity_signing_secret,
            ):
                return None
        return User(
            id=x_user_id,
            email=x_user_email,
            roles=[r.strip() for r in x_user_roles.split(",") if r.strip()] if x_user_roles else [],
        )

    def require_permission(self, namespace: str, relation: str, object_id: str, user: User) -> User:
        if not self.client.check(namespace, object_id, relation, user.id).allowed:
            try:
                from fastapi import HTTPException

                raise HTTPException(status_code=403, detail="Permission denied")
            except ImportError:
                raise PermissionError("Permission denied")
        return user


def autorix_dependency(security: AutorixSecurity):
    """Return a real FastAPI dependency reading trusted identity headers."""
    try:
        from fastapi import Header, HTTPException
    except ImportError as exc:
        raise RuntimeError("Install autorix[fastapi] to use this integration") from exc

    async def current_user(
        x_user_id: Optional[str] = Header(default=None),
        x_user_email: Optional[str] = Header(default=None),
        x_user_roles: Optional[str] = Header(default=None),
        x_user_scopes: Optional[str] = Header(default=None),
        x_autorix_identity_signature: Optional[str] = Header(default=None),
    ) -> User:
        user = security.get_current_user(
            x_user_id,
            x_user_email,
            x_user_roles,
            x_user_scopes,
            x_autorix_identity_signature,
        )
        if user is None:
            raise HTTPException(status_code=401, detail="Authentication required")
        return user

    return current_user
