"""FastAPI dependencies for fail-closed Autorix authorization."""
from __future__ import annotations
from typing import Optional
from .client import AutorixClient, User

class AutorixSecurity:
    def __init__(self, client: Optional[AutorixClient] = None): self.client = client or AutorixClient()
    def get_current_user(self, x_user_id: Optional[str] = None, x_user_email: Optional[str] = None, x_user_roles: Optional[str] = None) -> Optional[User]:
        if not x_user_id: return None
        return User(id=x_user_id, email=x_user_email, roles=[r.strip() for r in x_user_roles.split(",") if r.strip()] if x_user_roles else [])
    def require_permission(self, namespace: str, relation: str, object_id: str, user: User) -> User:
        if not self.client.check(namespace, object_id, relation, user.id).allowed:
            try:
                from fastapi import HTTPException
                raise HTTPException(status_code=403, detail="Permission denied")
            except ImportError: raise PermissionError("Permission denied")
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
    ) -> User:
        user = security.get_current_user(x_user_id, x_user_email, x_user_roles)
        if user is None:
            raise HTTPException(status_code=401, detail="Authentication required")
        return user
    return current_user
