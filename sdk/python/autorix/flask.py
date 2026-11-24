"""Flask integration; identity headers must be set by a trusted proxy."""
from functools import wraps
from typing import Optional

from .client import AutorixClient, User
from .identity import IDENTITY_SIGNATURE_HEADER, default_identity_signing_secret, verify_identity_headers


def current_user(identity_signing_secret: Optional[str] = None, require_signature: bool = False) -> Optional[User]:
    from flask import request

    user_id = request.headers.get("X-User-ID")
    if not user_id:
        return None
    email = request.headers.get("X-User-Email")
    roles = request.headers.get("X-User-Roles")
    scopes = request.headers.get("X-User-Scopes")
    secret = identity_signing_secret if identity_signing_secret is not None else default_identity_signing_secret()
    if require_signature or secret:
        if not verify_identity_headers(
            user_id,
            email,
            roles,
            scopes,
            request.headers.get(IDENTITY_SIGNATURE_HEADER),
            secret,
        ):
            return None
    return User(
        user_id,
        email,
        [r.strip() for r in (roles or "").split(",") if r.strip()],
    )


def require_permission(
    client: AutorixClient,
    namespace: str,
    relation: str,
    object_id=lambda: None,
    identity_signing_secret: Optional[str] = None,
    require_signature: bool = False,
):
    def decorator(view):
        @wraps(view)
        def wrapped(*args, **kwargs):
            from flask import abort

            user = current_user(
                identity_signing_secret=identity_signing_secret,
                require_signature=require_signature,
            )
            if user is None:
                abort(401)
            target = object_id() if callable(object_id) else object_id
            if not target or not client.check(namespace, target, relation, user.id).allowed:
                abort(403)
            return view(*args, **kwargs)

        return wrapped

    return decorator
